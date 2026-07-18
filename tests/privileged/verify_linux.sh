#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
    printf '%s\n' "Run this script as root." >&2
    exit 1
fi
if [ "$(uname -s)" != "Linux" ]; then
    printf '%s\n' "This verification requires Linux." >&2
    exit 1
fi
for command in openvpn ip sysctl curl pgrep pkill; do
    if ! command -v "$command" >/dev/null 2>&1; then
        printf '%s\n' "Missing required command: $command" >&2
        exit 1
    fi
done
if [ ! -c /dev/net/tun ]; then
    printf '%s\n' "/dev/net/tun is not available." >&2
    exit 1
fi

PROJECT_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
DATA_DIR=$(mktemp -d)
WEB_PORT=18787
PROXY_PORT=17928
TABLE=101
PID=""
DEFAULT_ROUTE_BEFORE=$(ip route show default)

cleanup() {
    exit_code=$?
    if [ "$exit_code" -ne 0 ] && [ -f "$DATA_DIR/service.out" ]; then
        printf '%s\n' "--- service output ---" >&2
        tail -n 200 "$DATA_DIR/service.out" >&2 || true
        printf '%s\n' "--- structured logs ---" >&2
        find "$DATA_DIR/logs" -type f -name '*.json' -exec tail -n 100 {} \; >&2 || true
    fi
    if [ -n "$PID" ]; then
        kill "$PID" >/dev/null 2>&1 || true
        wait "$PID" >/dev/null 2>&1 || true
    fi
    pkill -f "openvpn.*$DATA_DIR" >/dev/null 2>&1 || true
    ip rule del table "$TABLE" >/dev/null 2>&1 || true
    ip route flush table "$TABLE" >/dev/null 2>&1 || true
    if [ "${KEEP_DATA:-0}" != "1" ]; then
        rm -rf "$DATA_DIR"
    fi
}
trap cleanup EXIT INT TERM

cd "$PROJECT_ROOT"
export FREE_PROXY_ENVIRONMENT=test
export FREE_PROXY_DATA_DIR="$DATA_DIR"
export FREE_PROXY_DATABASE_URL="sqlite+aiosqlite:///$DATA_DIR/free-proxy.db"
export FREE_PROXY_WEB_HOST=127.0.0.1
export FREE_PROXY_WEB_PORT="$WEB_PORT"
export FREE_PROXY_ADMIN_AUTH_ENABLED=false
export FREE_PROXY_PROXY_HOST=127.0.0.1
export FREE_PROXY_PROXY_PORT="$PROXY_PORT"
export FREE_PROXY_POLICY_ROUTING_TABLE="$TABLE"
export FREE_PROXY_MAINTENANCE_ENABLED=false
export FREE_PROXY_DISCOVERY_LIMIT=30
export FREE_PROXY_INITIAL_CONNECT_TEST_LIMIT=10
export FREE_PROXY_OPENVPN_TEST_TIMEOUT_SECONDS=20
export FREE_PROXY_OPENVPN_CONNECT_TIMEOUT_SECONDS=45
export UV_PROJECT_ENVIRONMENT="$DATA_DIR/venv"

uv sync --frozen --dev
uv run free-proxy database-upgrade
uv run free-proxy serve >"$DATA_DIR/service.out" 2>&1 &
PID=$!

attempt=0
until curl -fsS "http://127.0.0.1:$WEB_PORT/api/v1/system/status" >/dev/null; do
    attempt=$((attempt + 1))
    if [ "$attempt" -ge 60 ]; then
        printf '%s\n' "Service did not become ready." >&2
        exit 1
    fi
    sleep 1
done

JOB_ID=$(curl -fsS -X POST "http://127.0.0.1:$WEB_PORT/api/v1/proxies/refresh" |
    uv run python -c 'import json,sys; print(json.load(sys.stdin)["id"])')

uv run python - "$WEB_PORT" "$JOB_ID" <<'PY'
import json
import sys
import time
import urllib.request

port, job_id = sys.argv[1:]
url = f"http://127.0.0.1:{port}/api/v1/jobs/{job_id}"
for _ in range(240):
    with urllib.request.urlopen(url, timeout=5) as response:
        job = json.load(response)
    if job["status"] == "succeeded":
        break
    if job["status"] in {"failed", "cancelled"}:
        raise SystemExit(f"Refresh job failed: {job}")
    time.sleep(1)
else:
    raise SystemExit("Refresh job timed out")
PY

ACTIVE_NODE=$(curl -fsS "http://127.0.0.1:$WEB_PORT/api/v1/gateway/status" |
    uv run python -c 'import json,sys; print(json.load(sys.stdin).get("active_node_id") or "")')
if [ -z "$ACTIVE_NODE" ]; then
    printf '%s\n' "No real OpenVPN exit was activated." >&2
    exit 1
fi

EXIT_CHECK_IP=$(uv run python -c \
    'import socket; print(next(item[4][0] for item in socket.getaddrinfo("api.ipify.org", 443, socket.AF_INET)))')
SOCKS_IP=$(curl -fsS --max-time 30 --proxy "socks5://127.0.0.1:$PROXY_PORT" \
    --connect-to "api.ipify.org:443:$EXIT_CHECK_IP:443" https://api.ipify.org)
HTTP_IP=$(curl -fsS --max-time 30 --proxy "http://127.0.0.1:$PROXY_PORT" \
    --connect-to "api.ipify.org:443:$EXIT_CHECK_IP:443" https://api.ipify.org)
if [ -z "$SOCKS_IP" ] || [ -z "$HTTP_IP" ]; then
    printf '%s\n' "Proxy exit IP verification failed." >&2
    exit 1
fi
if [ "$SOCKS_IP" != "$HTTP_IP" ]; then
    printf '%s\n' "SOCKS5 and HTTP proxy exits differ: $SOCKS_IP vs $HTTP_IP" >&2
    exit 1
fi
printf '%s\n' "Verified SOCKS5 and HTTP exit IP: $SOCKS_IP"

DEFAULT_ROUTE_AFTER=$(ip route show default)
if [ "$DEFAULT_ROUTE_BEFORE" != "$DEFAULT_ROUTE_AFTER" ]; then
    printf '%s\n' "The host default route changed." >&2
    exit 1
fi
if ! ip address show tun0 >/dev/null; then
    printf '%s\n' "tun0 is missing after activation." >&2
    exit 1
fi
if ! ip rule show | grep -q "lookup $TABLE"; then
    printf '%s\n' "Policy rule for table $TABLE is missing." >&2
    ip rule show >&2
    exit 1
fi
if ! ip route show table "$TABLE" | grep -q '^default .*dev tun0'; then
    printf '%s\n' "Default policy route through tun0 is missing." >&2
    ip route show table "$TABLE" >&2
    exit 1
fi
RP_FILTER=$(sysctl -n net.ipv4.conf.all.rp_filter)
if [ "$RP_FILTER" -ne 2 ]; then
    printf '%s\n' "rp_filter was not set to 2: $RP_FILTER" >&2
    exit 1
fi

kill "$PID"
wait "$PID" || true
PID=""
sleep 2
if pgrep -f "openvpn.*$DATA_DIR" >/dev/null; then
    printf '%s\n' "OpenVPN process remained after shutdown." >&2
    exit 1
fi
if ip rule show | grep -q "lookup $TABLE"; then
    printf '%s\n' "Policy rule remained after shutdown." >&2
    exit 1
fi
if ip route show table "$TABLE" | grep -q .; then
    printf '%s\n' "Policy route remained after shutdown." >&2
    exit 1
fi

printf '%s\n' "Linux privileged verification passed with exit IP $SOCKS_IP and node $ACTIVE_NODE"
