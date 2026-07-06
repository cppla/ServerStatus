#!/bin/sh
set -eu

: "${CONFIG_PATH:=/ServerStatus/server/config.json}"
: "${WEB_DIR:=/usr/share/nginx/html}"
: "${SERGATE_PID_FILE:=/tmp/serverstatus-sergate.pid}"
: "${ADMIN_API_BIND:=127.0.0.1}"
: "${ADMIN_API_PORT:=35602}"

STOPPING=0
SERGATE_PID=""
API_PID=""

stop_all() {
  STOPPING=1
  if [ -n "$SERGATE_PID" ] && kill -0 "$SERGATE_PID" 2>/dev/null; then
    kill -TERM "$SERGATE_PID" 2>/dev/null || true
  fi
  if [ -n "$API_PID" ] && kill -0 "$API_PID" 2>/dev/null; then
    kill -TERM "$API_PID" 2>/dev/null || true
  fi
  nginx -s quit 2>/dev/null || true
}

trap 'stop_all; exit 0' INT TERM QUIT

mkdir -p "$WEB_DIR/json"
nginx

CONFIG_PATH="$CONFIG_PATH" \
STATS_PATH="${STATS_PATH:-$WEB_DIR/json/stats.json}" \
SERGATE_PID_FILE="$SERGATE_PID_FILE" \
ADMIN_API_BIND="$ADMIN_API_BIND" \
ADMIN_API_PORT="$ADMIN_API_PORT" \
ADMIN_TOKEN="${ADMIN_TOKEN:-}" \
ADMIN_CORS_ORIGIN="${ADMIN_CORS_ORIGIN:-}" \
  python3 /ServerStatus/server/manage_api.py &
API_PID="$!"
if [ -n "${ADMIN_TOKEN:-}" ]; then
  echo "management API enabled on ${ADMIN_API_BIND}:${ADMIN_API_PORT}"
else
  echo "management API running in read-only discovery mode; set ADMIN_TOKEN to enable writes"
fi

while [ "$STOPPING" -eq 0 ]; do
  /ServerStatus/server/sergate --config="$CONFIG_PATH" --web-dir="$WEB_DIR" &
  SERGATE_PID="$!"
  echo "$SERGATE_PID" > "$SERGATE_PID_FILE"
  set +e
  wait "$SERGATE_PID"
  STATUS="$?"
  set -e
  rm -f "$SERGATE_PID_FILE"
  if [ "$STOPPING" -eq 1 ]; then
    exit "$STATUS"
  fi
  echo "sergate exited with status ${STATUS}; restarting in 1s"
  sleep 1
done
