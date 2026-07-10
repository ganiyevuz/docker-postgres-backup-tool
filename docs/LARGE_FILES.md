# Sending Backups Larger Than 50 MB

The official Telegram Bot API (`https://api.telegram.org`) caps uploads at **50 MB**. To send larger backups, pick one of two routes.

## Route A — Built-in MTProto uploader (recommended, no extra service)

The image bundles `tg-upload`, a static binary that uploads over MTProto (up to **2 GB**) directly — no sidecar container. Backups ≤ 50 MB still go through the normal Bot API path; only larger files use MTProto.

**This works out of the box.** The image ships with a shared Telegram app, so large-file upload needs no setup beyond `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID`. The shared app only identifies the app to Telegram — your bot token authenticates and backups go to your own chat, so your data is unaffected.

For full independence (recommended), register your own app at <https://my.telegram.org/apps> and set both on the `backup` service:
   - `TELEGRAM_API_ID`
   - `TELEGRAM_API_HASH`

To disable the shared default and require your own, set `TELEGRAM_USE_DEFAULT_API=FALSE`. Leave `TELEGRAM_API_URL` unset (defaults to the official API).

See [`examples/docker-compose.large-files-mtproto.yml`](https://github.com/ganiyevuz/backupgram/blob/main/examples/docker-compose.large-files-mtproto.yml).

**Targets:** the bot must be a member of the destination. Supergroups/channels (`-100…` ids) and basic groups work. Sending to an individual user requires that user to have started the bot.

**Multiple chats:** `TELEGRAM_CHAT_ID` accepts a comma-separated list (e.g. `-1001111111111,12345678`). The backup is uploaded once and delivered to every chat (`file_id` reuse for ≤50 MB, a single MTProto upload reused for larger files). `TELEGRAM_THREAD_ID` is honoured only when exactly one chat is configured.

`TELEGRAM_API_ID` / `TELEGRAM_API_HASH` also support Docker secrets via `TELEGRAM_API_ID_FILE` / `TELEGRAM_API_HASH_FILE`.

## Route B — Self-hosted Bot API server

Run the official `telegram-bot-api` daemon as a sidecar and point `TELEGRAM_API_URL` at it. The existing `curl` path then handles files up to 2 GB unchanged. This is heavier (an extra service + volume) but keeps everything on the Bot API. The sidecar is an upstream image, so it needs its own `api_id`/`api_hash` — the image's shared default applies only to the built-in MTProto uploader in Route A.

See [`examples/docker-compose.large-files-server.yml`](https://github.com/ganiyevuz/backupgram/blob/main/examples/docker-compose.large-files-server.yml).

## Forcing a transport with `TELEGRAM_UPLOAD_METHOD`

By default delivery is `smart` — it picks the transport automatically by file size and configuration. Set `TELEGRAM_UPLOAD_METHOD` to force one:

- `smart` (default) — ≤50 MB via the Bot API; larger files via a self-hosted server (if `TELEGRAM_API_URL` is custom) or MTProto (using your `TELEGRAM_API_ID`/`TELEGRAM_API_HASH`, or the shared default when they are unset).
- `botapi` — always the Bot API (`curl`) against `TELEGRAM_API_URL`. On the official API, files >50 MB are kept locally with a warning (never silently switched to MTProto).
- `mtproto` — always the bundled `tg-upload` binary, for files of any size. Uses your `TELEGRAM_API_ID`/`TELEGRAM_API_HASH` or the shared default (validated at startup).

An explicitly chosen method is never silently overridden; if it cannot deliver a file, the backup is kept locally and the run continues.

## Which to choose?

| | Route A (MTProto binary) | Route B (self-hosted server) |
|---|---|---|
| Extra service | none | `telegram-bot-api` container |
| Max size | 2 GB | 2 GB |
| Needs `api_id`/`api_hash` | optional (shared default ships) | yes (its own) |
| Best for | most users | those already running a Bot API server |

## Disaster recovery: restoring from Telegram

Every backup message's caption includes its own restore id, e.g. `🔖 Restore ID: 4521`. If the `BACKUP_DIR` volume is lost, you can restore straight from Telegram:

```sh
docker exec -it my-backup restore --from-telegram 4521 [--chat <chat_id>] [target_db]
```

`tg-upload` downloads the document for that message over MTProto (preserving the original filename, so GPG/format detection works), and the file is then restored through the normal pipeline. For multi-chat delivery, message ids are per-chat — read the id from the caption in the chat you intend to restore from, and pass `--chat` if it is not the first configured chat.
