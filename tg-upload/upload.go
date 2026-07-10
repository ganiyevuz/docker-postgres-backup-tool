package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

// runUpload implements the default (no-subcommand) behaviour: upload a file and
// fan it out to one or more chats. args is os.Args[1:] (no subcommand consumed).
func runUpload(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("tg-upload", flag.ContinueOnError)
	file := fs.String("file", "", "path to the file to upload (required)")
	chat := fs.String("chat", "", "target chat id (required)")
	thread := fs.Int("thread", 0, "message thread / forum topic id (0 = none)")
	caption := fs.String("caption", "", "document caption")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *file == "" || *chat == "" {
		return errors.New("--file and --chat are required")
	}

	apiID, apiHash, botToken, err := credsFromEnv()
	if err != nil {
		return err
	}

	targets, err := splitChats(*chat)
	if err != nil {
		return err
	}

	f, err := os.Open(*file)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	client := telegram.NewClient(apiID, apiHash, telegram.Options{})
	return client.Run(ctx, func(ctx context.Context) error {
		if _, err := client.Auth().Bot(ctx, botToken); err != nil {
			return fmt.Errorf("bot auth: %w", err)
		}

		api := tg.NewClient(client)
		baseName := filepath.Base(*file)

		inputFile, err := uploader.NewUploader(api).
			WithThreads(4).
			WithPartSize(512*1024).
			WithProgress(newProgressLogger()).
			Upload(ctx, uploader.NewUpload(baseName, f, stat.Size()))
		if err != nil {
			return fmt.Errorf("upload: %w", err)
		}

		var docOpts []message.StyledTextOption
		if *caption != "" {
			docOpts = append(docOpts, styling.Plain(*caption))
		}

		sender := message.NewSender(api)
		useThread := *thread != 0 && len(targets) == 1

		sent := 0
		for _, t := range targets {
			rb := sender.To(t.peer)
			var b *message.Builder
			if useThread {
				b = rb.Reply(*thread)
			} else {
				b = rb.CloneBuilder()
			}
			doc := message.UploadedDocument(inputFile, docOpts...).
				Filename(baseName).
				ForceFile(true)
			upd, err := b.Media(ctx, doc)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tg-upload: ⚠️ failed to send to %s: %v\n", t.raw, err)
				continue
			}
			sent++
			fmt.Fprintf(os.Stderr, "tg-upload: ✅ sent to %s\n", t.raw)
			editCaptionAfterSend(ctx, api, t.peer, upd, *caption)
		}
		if sent == 0 {
			return fmt.Errorf("upload succeeded but all %d send(s) failed", len(targets))
		}
		return nil
	})
}

// defaultAPIID and defaultAPIHash are the shared fallback Telegram app
// credentials, compiled in via -ldflags "-X main.defaultAPIID=... -X
// main.defaultAPIHash=...". They are empty unless the image was built with the
// tg_default_api build secret. They only identify the *app* to Telegram; the
// bot token still authenticates and operator data is never exposed.
var (
	defaultAPIID   string
	defaultAPIHash string
)

// credsFromEnv reads and validates the Telegram MTProto credentials, falling
// back to the compiled-in shared defaults when the operator set neither
// TELEGRAM_API_ID nor TELEGRAM_API_HASH.
func credsFromEnv() (apiID int, apiHash, botToken string, err error) {
	rawAPIID := os.Getenv("TELEGRAM_API_ID")
	apiHash = os.Getenv("TELEGRAM_API_HASH")
	if rawAPIID == "" && apiHash == "" && defaultAPIID != "" && defaultAPIHash != "" {
		rawAPIID = defaultAPIID
		apiHash = defaultAPIHash
		fmt.Fprintln(os.Stderr, "tg-upload: ℹ️ using backupgram's shared Telegram app; set your own TELEGRAM_API_ID/TELEGRAM_API_HASH (https://my.telegram.org/apps) to be fully independent.")
	}
	if rawAPIID == "" {
		return 0, "", "", errors.New("TELEGRAM_API_ID must be set")
	}
	apiID, err = strconv.Atoi(rawAPIID)
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid TELEGRAM_API_ID %q: %w", rawAPIID, err)
	}
	botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if apiHash == "" || botToken == "" {
		return 0, "", "", errors.New("TELEGRAM_API_HASH and TELEGRAM_BOT_TOKEN must be set")
	}
	return apiID, apiHash, botToken, nil
}
