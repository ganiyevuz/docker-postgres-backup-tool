#!/usr/bin/env bash
set -Eeo pipefail

# Prevalidate configuration (don't source)
if [ "${VALIDATE_ON_START}" = "TRUE" ]; then
  echo "Running pre-validation script..."
  if ! /env.sh; then
    echo "Error: Validation failed, aborting." >&2
    exit 1
  fi
fi

# Initial background backup
EXTRA_ARGS=""
if [ "${BACKUP_ON_START}" = "TRUE" ]; then
  EXTRA_ARGS="-i"
fi

# Running the go-cron job
echo "Starting cron job with schedule: $SCHEDULE and health check port: $HEALTHCHECK_PORT"
if ! exec /usr/local/bin/go-cron -s "$SCHEDULE" -p "$HEALTHCHECK_PORT" $EXTRA_ARGS -- /backup.sh; then
  echo "Error: go-cron job failed to start." >&2
  exit 1
fi
