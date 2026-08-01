-- +goose Up
CREATE TABLE app_metadata (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE admin_settings (
    id                  INTEGER PRIMARY KEY CHECK (id = 1),
    username            TEXT NOT NULL DEFAULT '',
    password_hash       TEXT NOT NULL DEFAULT '',
    secret_path         TEXT NOT NULL DEFAULT '',
    session_ttl_seconds INTEGER NOT NULL DEFAULT 2592000,
    web_port            INTEGER NOT NULL DEFAULT 39527,
    web_external_access INTEGER NOT NULL DEFAULT 1
);
INSERT INTO admin_settings (id) VALUES (1);

CREATE TABLE proxy_settings (
    id                      INTEGER PRIMARY KEY CHECK (id = 1),
    enabled                 INTEGER NOT NULL DEFAULT 1,
    port                    INTEGER NOT NULL DEFAULT 9527,
    username                TEXT NOT NULL DEFAULT '',
    password_hash           TEXT NOT NULL DEFAULT '',
    external_access         INTEGER NOT NULL DEFAULT 0,
    max_connections         INTEGER NOT NULL DEFAULT 256,
    connect_timeout_seconds REAL NOT NULL DEFAULT 20,
    idle_timeout_seconds    REAL NOT NULL DEFAULT 120,
    dns_server              TEXT NOT NULL DEFAULT '8.8.8.8'
);
INSERT INTO proxy_settings (id) VALUES (1);

CREATE TABLE discovery_settings (
    id                    INTEGER PRIMARY KEY CHECK (id = 1),
    vpngate_api_url       TEXT NOT NULL DEFAULT 'https://www.vpngate.net/api/iphone/',
    discovery_limit       INTEGER NOT NULL DEFAULT 300,
    request_timeout_secs  REAL NOT NULL DEFAULT 15,
    ip_info_api_url       TEXT NOT NULL DEFAULT 'http://ip-api.com/batch?lang=zh-CN&fields=status,message,query,country,regionName,city,isp,org,as,asname,proxy,hosting,mobile',
    ip_info_cache_seconds INTEGER NOT NULL DEFAULT 604800
);
INSERT INTO discovery_settings (id) VALUES (1);

CREATE TABLE maintenance_settings (
    id                               INTEGER PRIMARY KEY CHECK (id = 1),
    enabled                          INTEGER NOT NULL DEFAULT 1,
    maintenance_interval_seconds     REAL NOT NULL DEFAULT 10800,
    health_check_interval_seconds    REAL NOT NULL DEFAULT 30,
    active_ping_interval_seconds     REAL NOT NULL DEFAULT 10,
    disconnected_retry_seconds       REAL NOT NULL DEFAULT 30,
    max_probe_concurrency            INTEGER NOT NULL DEFAULT 5,
    initial_connect_test_limit       INTEGER NOT NULL DEFAULT 10,
    manual_test_node_limit           INTEGER NOT NULL DEFAULT 5,
    openvpn_test_timeout_seconds     REAL NOT NULL DEFAULT 15,
    openvpn_connect_timeout_seconds  REAL NOT NULL DEFAULT 35,
    invalid_backoff_seconds          INTEGER NOT NULL DEFAULT 1800,
    stale_node_grace_seconds         INTEGER NOT NULL DEFAULT 604800
);
INSERT INTO maintenance_settings (id) VALUES (1);

CREATE TABLE network_settings (
    id                             INTEGER PRIMARY KEY CHECK (id = 1),
    dns_repair_enabled             INTEGER NOT NULL DEFAULT 0,
    dns_repair_servers             TEXT NOT NULL DEFAULT '1.1.1.1,8.8.8.8',
    routing_setup_retries          INTEGER NOT NULL DEFAULT 3,
    routing_retry_interval_seconds REAL NOT NULL DEFAULT 1,
    routing_strict_rp_filter       INTEGER NOT NULL DEFAULT 0
);
INSERT INTO network_settings (id) VALUES (1);

-- +goose Down
DROP TABLE network_settings;
DROP TABLE maintenance_settings;
DROP TABLE discovery_settings;
DROP TABLE proxy_settings;
DROP TABLE admin_settings;
DROP TABLE app_metadata;
DROP TABLE app_metadata;
