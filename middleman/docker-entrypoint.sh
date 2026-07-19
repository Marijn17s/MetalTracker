#!/bin/sh
set -eu

DB_PATH="${DB_PATH:-/data/prices.db}"
DATA_DIR="$(dirname "$DB_PATH")"

mkdir -p "$DATA_DIR"
chown -R middleman:middleman "$DATA_DIR"

exec su-exec middleman /usr/local/bin/middleman
