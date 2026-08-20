#!/bin/sh
set -eu

ROOT=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
TEST_DIR=$(mktemp -d "${TMPDIR:-/tmp}/serverstatus-webui.XXXXXX")
SERVER_PID=""

# Invoked by trap.
# shellcheck disable=SC2329
cleanup() {
  status=$?
  trap - EXIT INT TERM HUP
  if [ -n "$SERVER_PID" ]; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  rm -rf "$TEST_DIR"
  exit "$status"
}
trap cleanup EXIT INT TERM HUP

cp "$ROOT/tests/fixtures/config.json" "$TEST_DIR/config.json"
mkdir -p "$TEST_DIR/data"

(
  cd "$ROOT/server"
  go build -trimpath -o "$TEST_DIR/serverstatus" .
)

ADMIN_TOKEN=test-token "$TEST_DIR/serverstatus" \
  --config="$TEST_DIR/config.json" \
  --stats="$TEST_DIR/data/stats.json" \
  --web-dir="$ROOT/web" \
  --http=127.0.0.1:18080 \
  --agent=127.0.0.1:35699 &
SERVER_PID=$!
wait "$SERVER_PID"
