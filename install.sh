#!/bin/sh
set -eu

# Free Proxy bootstrap: download the Go binary for this architecture and hand
# off to `free-proxy install`, which handles everything else (system
# dependencies, environment file, init service). Re-run to update.

REPO="${FREE_PROXY_REPO:-masteralanlab/free-proxy}"
RELEASE="${FREE_PROXY_RELEASE:-latest}"
BIN="/usr/local/bin/free-proxy"

[ "$(id -u)" -eq 0 ] || { echo "This script must run as root." >&2; exit 1; }

case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

if [ "$RELEASE" = "latest" ]; then
    url="https://github.com/$REPO/releases/latest/download/free-proxy-linux-$arch"
else
    url="https://github.com/$REPO/releases/download/$RELEASE/free-proxy-linux-$arch"
fi

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
echo "Downloading $url"
curl -fL "$url" -o "$tmp"
install -m 0755 "$tmp" "$BIN"

exec "$BIN" install
