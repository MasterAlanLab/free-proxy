#!/bin/sh
set -eu

APP_NAME="free-proxy"
INSTALL_DIR="${FREE_PROXY_INSTALL_DIR:-/opt/free-proxy}"
DATA_DIR="${FREE_PROXY_DATA_DIR:-/var/lib/free-proxy}"
CONFIG_DIR="${FREE_PROXY_CONFIG_DIR:-/etc/free-proxy}"
ENV_FILE="$CONFIG_DIR/free-proxy.env"
REPO_URL="${FREE_PROXY_REPO_URL:-https://github.com/masteralanlab/free-proxy.git}"
BRANCH="${FREE_PROXY_BRANCH:-main}"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ACTION="${1:-install}"

require_root() {
    if [ "$(id -u)" -ne 0 ]; then
        printf '%s\n' "This command must run as root." >&2
        exit 1
    fi
}

install_packages() {
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update
        DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl git iproute2 openvpn
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y ca-certificates curl git iproute openvpn
    elif command -v yum >/dev/null 2>&1; then
        yum install -y ca-certificates curl git iproute openvpn
    elif command -v apk >/dev/null 2>&1; then
        apk add --no-cache ca-certificates curl git iproute2 openvpn
    else
        printf '%s\n' "Unsupported package manager; install curl, git, iproute2 and OpenVPN first." >&2
        exit 1
    fi
}

install_uv() {
    if command -v uv >/dev/null 2>&1; then
        return
    fi
    curl -LsSf https://astral.sh/uv/install.sh | env UV_INSTALL_DIR=/usr/local/bin sh
}

sync_source() {
    temporary=""
    source_dir="$SCRIPT_DIR"
    if [ ! -f "$source_dir/pyproject.toml" ] || [ ! -d "$source_dir/src/free_proxy" ]; then
        temporary=$(mktemp -d)
        git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$temporary/source"
        source_dir="$temporary/source"
    fi
    mkdir -p "$INSTALL_DIR"
    find "$INSTALL_DIR" -mindepth 1 -maxdepth 1 ! -name .venv -exec rm -rf {} +
    tar --exclude=.git --exclude=.venv --exclude=free_proxy_data -C "$source_dir" -cf - . |
        tar -C "$INSTALL_DIR" -xf -
    if [ -n "$temporary" ]; then
        rm -rf "$temporary"
    fi
}

write_environment() {
    mkdir -p "$CONFIG_DIR" "$DATA_DIR"
    if [ ! -f "$ENV_FILE" ]; then
        cat >"$ENV_FILE" <<EOF
FREE_PROXY_ENVIRONMENT=production
FREE_PROXY_DATA_DIR=$DATA_DIR
FREE_PROXY_WEB_HOST=127.0.0.1
FREE_PROXY_WEB_PORT=8787
FREE_PROXY_PROXY_HOST=127.0.0.1
FREE_PROXY_PROXY_PORT=9527
FREE_PROXY_PROXY_ENABLED=true
FREE_PROXY_MAINTENANCE_ENABLED=true
FREE_PROXY_DNS_REPAIR_ENABLED=false
EOF
        chmod 600 "$ENV_FILE"
    fi
}

install_runtime() {
    cd "$INSTALL_DIR"
    uv sync --frozen --no-dev
    set -a
    . "$ENV_FILE"
    set +a
    "$INSTALL_DIR/.venv/bin/free-proxy" database-upgrade
    "$INSTALL_DIR/.venv/bin/free-proxy" preflight
    "$INSTALL_DIR/.venv/bin/free-proxy" credentials
}

install_service() {
    if command -v systemctl >/dev/null 2>&1; then
        sed \
            -e "s|/opt/free-proxy|$INSTALL_DIR|g" \
            -e "s|/etc/free-proxy/free-proxy.env|$ENV_FILE|g" \
            "$INSTALL_DIR/deploy/free-proxy.service" >/etc/systemd/system/free-proxy.service
        systemctl daemon-reload
        systemctl enable free-proxy.service
    elif command -v rc-service >/dev/null 2>&1; then
        sed \
            -e "s|/opt/free-proxy|$INSTALL_DIR|g" \
            -e "s|/etc/free-proxy/free-proxy.env|$ENV_FILE|g" \
            "$INSTALL_DIR/deploy/free-proxy.openrc" >/etc/init.d/free-proxy
        chmod 755 /etc/init.d/free-proxy
        rc-update add free-proxy default
    else
        printf '%s\n' "Neither systemd nor OpenRC was detected." >&2
        exit 1
    fi
    ln -sf "$INSTALL_DIR/.venv/bin/free-proxy" /usr/local/bin/free-proxy
    ln -sf "$INSTALL_DIR/install.sh" /usr/local/bin/free-proxy-manage
}

service_action() {
    action="$1"
    if command -v systemctl >/dev/null 2>&1; then
        systemctl "$action" free-proxy.service
    else
        rc-service free-proxy "$action"
    fi
}

show_logs() {
    if command -v journalctl >/dev/null 2>&1; then
        exec journalctl -u free-proxy.service -f
    fi
    latest=$(find "$DATA_DIR/logs" -type f -name '*.json' 2>/dev/null | sort | tail -n 1)
    if [ -n "$latest" ]; then
        exec tail -f "$latest"
    fi
    printf '%s\n' "No log file is available yet."
}

uninstall_app() {
    service_action stop >/dev/null 2>&1 || true
    if command -v systemctl >/dev/null 2>&1; then
        systemctl disable free-proxy.service >/dev/null 2>&1 || true
        rm -f /etc/systemd/system/free-proxy.service
        systemctl daemon-reload
    else
        rc-update del free-proxy default >/dev/null 2>&1 || true
        rm -f /etc/init.d/free-proxy
    fi
    rm -f /usr/local/bin/free-proxy /usr/local/bin/free-proxy-manage
    rm -rf "$INSTALL_DIR" "$CONFIG_DIR"
    if [ "${PURGE_DATA:-0}" = "1" ]; then
        rm -rf "$DATA_DIR"
    fi
}

case "$ACTION" in
    install)
        require_root
        install_packages
        install_uv
        sync_source
        write_environment
        install_runtime
        install_service
        service_action restart
        printf '%s\n' "Free Proxy installed. Run: free-proxy-manage credentials"
        ;;
    update)
        require_root
        install_uv
        sync_source
        write_environment
        install_runtime
        install_service
        service_action restart
        ;;
    uninstall)
        require_root
        uninstall_app
        ;;
    start|stop|restart|status)
        require_root
        service_action "$ACTION"
        ;;
    logs)
        require_root
        show_logs
        ;;
    credentials)
        set -a
        . "$ENV_FILE"
        set +a
        exec "$INSTALL_DIR/.venv/bin/free-proxy" credentials
        ;;
    *)
        printf '%s\n' "Usage: $0 {install|update|uninstall|start|stop|restart|status|logs|credentials}" >&2
        exit 2
        ;;
esac
