#!/bin/sh
set -eu

ROOT=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
SERVER_IMAGE=${SERVER_IMAGE:-serverstatus-server:test}
CLIENT_IMAGE=${CLIENT_IMAGE:-serverstatus-client:test}
SUFFIX=$$
NETWORK="serverstatus-smoke-$SUFFIX"
SERVER_NAME="serverstatus-smoke-server-$SUFFIX"
CLIENT_NAME="serverstatus-smoke-client-$SUFFIX"
mkdir -p "$ROOT/output"
TEST_DIR=$(mktemp -d "$ROOT/output/docker-smoke.XXXXXX")

# Invoked by trap.
# shellcheck disable=SC2329
cleanup() {
  status=$?
  trap - EXIT INT TERM HUP
  if [ "$status" -ne 0 ]; then
    docker logs "$SERVER_NAME" 2>/dev/null || true
    docker logs "$CLIENT_NAME" 2>/dev/null || true
  fi
  docker rm -f "$CLIENT_NAME" "$SERVER_NAME" >/dev/null 2>&1 || true
  docker network rm "$NETWORK" >/dev/null 2>&1 || true
  rm -rf "$TEST_DIR"
  exit "$status"
}
trap cleanup EXIT INT TERM HUP

cp "$ROOT/tests/fixtures/config.json" "$TEST_DIR/config.json"
mkdir -p "$TEST_DIR/data"

docker run --rm --entrypoint python3 "$CLIENT_IMAGE" -c \
  'import platform, psutil; assert psutil.cpu_count(); print(platform.machine(), psutil.__version__)'

docker_os=$(docker info --format '{{.OperatingSystem}}')
if printf '%s' "$docker_os" | grep -q 'Docker Desktop'; then
  mode=bridge
  docker network create "$NETWORK" >/dev/null
  docker run -d \
    --name "$SERVER_NAME" \
    --network "$NETWORK" \
    -e ADMIN_TOKEN=test-token \
    -v "$TEST_DIR/config.json:/app/config/config.json" \
    -v "$TEST_DIR/data:/app/data" \
    "$SERVER_IMAGE" >/dev/null
else
  mode=host
  docker run -d \
    --name "$SERVER_NAME" \
    -e ADMIN_TOKEN=test-token \
    -v "$TEST_DIR/config.json:/app/config/config.json" \
    -v "$TEST_DIR/data:/app/data" \
    -p 127.0.0.1::80 \
    -p 127.0.0.1::35601 \
    "$SERVER_IMAGE" >/dev/null
  HTTP_PORT=$(docker port "$SERVER_NAME" 80/tcp | awk -F: 'NR == 1 { print $NF }')
  AGENT_PORT=$(docker port "$SERVER_NAME" 35601/tcp | awk -F: 'NR == 1 { print $NF }')
  if [ -z "$HTTP_PORT" ] || [ -z "$AGENT_PORT" ] || [ "$HTTP_PORT" = "$AGENT_PORT" ]; then
    echo "could not determine distinct published server ports" >&2
    exit 1
  fi
fi

attempt=0
until docker exec "$SERVER_NAME" wget -q -O /dev/null http://127.0.0.1/api/health; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 30 ]; then
    echo "server container did not become healthy" >&2
    exit 1
  fi
  sleep 1
done

if [ "$mode" = host ]; then
  docker run -d \
    --name "$CLIENT_NAME" \
    --network host \
    --pid host \
    -e SERVER=127.0.0.1 \
    -e PORT="$AGENT_PORT" \
    -e USER=s01 \
    -e PASSWORD=fixture-password \
    -e CLIENT=psutil \
    -e PYTHONUNBUFFERED=1 \
    -e INTERVAL=1 \
    -e CU=127.0.0.1 \
    -e CT=127.0.0.1 \
    -e CM=127.0.0.1 \
    -e PROBEPORT="$HTTP_PORT" \
    "$CLIENT_IMAGE" >/dev/null
else
  echo "Docker Desktop: host networking unavailable; using bridge fallback locally."
  docker run -d \
    --name "$CLIENT_NAME" \
    --network "$NETWORK" \
    --pid host \
    -e SERVER="$SERVER_NAME" \
    -e PORT=35601 \
    -e USER=s01 \
    -e PASSWORD=fixture-password \
    -e CLIENT=psutil \
    -e PYTHONUNBUFFERED=1 \
    -e INTERVAL=1 \
    -e CU="$SERVER_NAME" \
    -e CT="$SERVER_NAME" \
    -e CM="$SERVER_NAME" \
    -e PROBEPORT=80 \
    "$CLIENT_IMAGE" >/dev/null
fi

attempt=0
while [ "$attempt" -lt 45 ]; do
  stats=$(docker exec "$SERVER_NAME" wget -q -O - http://127.0.0.1/json/stats.json 2>/dev/null || true)
  if printf '%s' "$stats" | python3 -c '
import json
import sys

document = json.load(sys.stdin)
node = next(item for item in document.get("servers", []) if item.get("name") == "fixture-node")
assert node.get("online4") or node.get("online6")
assert int(node.get("cpu_cores") or 0) > 0
assert str(node.get("cpu_model") or "").strip()
assert node.get("os") == "alpine"
assert int(node.get("network_in") or 0) > 0
assert int(node.get("network_out") or 0) > 0
' 2>/dev/null; then
    echo "Docker smoke passed ($mode): client authenticated and reported live metrics."
    exit 0
  fi
  attempt=$((attempt + 1))
  sleep 1
done

echo "client did not report live metrics before timeout" >&2
exit 1
