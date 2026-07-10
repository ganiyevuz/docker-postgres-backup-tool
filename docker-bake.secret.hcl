# CI-only override — adds the shared-default Telegram build secret to every image
# target. Used ONLY in CI:
#
#   docker buildx bake -f docker-bake.hcl -f docker-bake.secret.hcl <target>
#
# The workflow first writes docker/default-telegram-api (gitignored) from the
# TG_DEFAULT_API_ID / TG_DEFAULT_API_HASH repo secrets. The base docker-bake.hcl
# carries no secret, so a plain local `docker buildx bake` builds with no default
# baked in (env.sh and tg-upload degrade gracefully). Do not add values here —
# this file only references the (gitignored) source path.

target "debian" {
	secret = ["id=tg_default_api,src=docker/default-telegram-api"]
}

target "alpine" {
	secret = ["id=tg_default_api,src=docker/default-telegram-api"]
}
