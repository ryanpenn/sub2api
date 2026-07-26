#!/bin/sh
set -eu

# Explicit commands are used by image verification and do not need runtime secrets.
if [ "$#" -gt 0 ]; then
    exec "$@"
fi

redis_password_file=/run/secrets/caddy_redis_password
storage_key_file=/run/secrets/caddy_storage_encryption_key

if [ ! -r "$redis_password_file" ] || [ ! -r "$storage_key_file" ]; then
    echo "required Caddy storage secrets are not readable" >&2
    exit 1
fi

CADDY_REDIS_PASSWORD=$(cat "$redis_password_file")
CADDY_STORAGE_ENCRYPTION_KEY=$(cat "$storage_key_file")

if [ -z "$CADDY_REDIS_PASSWORD" ]; then
    echo "Caddy Redis password must not be empty" >&2
    exit 1
fi
if [ "${#CADDY_STORAGE_ENCRYPTION_KEY}" -ne 32 ]; then
    echo "Caddy storage encryption key must contain exactly 32 characters" >&2
    exit 1
fi

export CADDY_REDIS_PASSWORD CADDY_STORAGE_ENCRYPTION_KEY
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
