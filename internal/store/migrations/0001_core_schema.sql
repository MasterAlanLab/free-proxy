-- +goose Up
CREATE TABLE proxy_nodes (
    id                   TEXT PRIMARY KEY,
    provider             TEXT NOT NULL DEFAULT '',
    provider_node_id     TEXT NOT NULL DEFAULT '',
    provider_identity    TEXT NOT NULL DEFAULT '',
    country              TEXT NOT NULL DEFAULT '',
    country_code         TEXT NOT NULL DEFAULT '',
    host_name            TEXT NOT NULL DEFAULT '',
    ip_address           TEXT NOT NULL DEFAULT '',
    remote_host          TEXT NOT NULL DEFAULT '',
    remote_port          INTEGER NOT NULL DEFAULT 0,
    transport            TEXT NOT NULL DEFAULT 'unknown',
    ip_type              TEXT NOT NULL DEFAULT 'unknown',
    owner                TEXT NOT NULL DEFAULT '',
    asn                  TEXT NOT NULL DEFAULT '',
    as_name              TEXT NOT NULL DEFAULT '',
    location             TEXT NOT NULL DEFAULT '',
    quality              TEXT NOT NULL DEFAULT '',
    status               TEXT NOT NULL DEFAULT 'discovered',
    source_score         INTEGER NOT NULL DEFAULT 0,
    source_ping_ms       INTEGER NOT NULL DEFAULT 0,
    source_speed_bps     INTEGER NOT NULL DEFAULT 0,
    source_sessions      INTEGER NOT NULL DEFAULT 0,
    latency_ms           INTEGER NOT NULL DEFAULT 0,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    success_count        INTEGER NOT NULL DEFAULT 0,
    failure_count        INTEGER NOT NULL DEFAULT 0,
    config_text          TEXT NOT NULL DEFAULT '',
    fetched_at           TEXT NOT NULL,
    last_probed_at       TEXT,
    last_success_at      TEXT,
    ip_info_updated_at   TEXT,
    cooldown_until       TEXT,
    last_seen_at         TEXT,
    source_present       INTEGER NOT NULL DEFAULT 1,
    UNIQUE (provider, provider_identity)
);
CREATE INDEX idx_nodes_provider ON proxy_nodes(provider);
CREATE INDEX idx_nodes_country  ON proxy_nodes(country);
CREATE INDEX idx_nodes_ip_type  ON proxy_nodes(ip_type);
CREATE INDEX idx_nodes_status   ON proxy_nodes(status);
CREATE INDEX idx_nodes_present  ON proxy_nodes(source_present);

CREATE TABLE ip_info_cache (
    ip_address TEXT PRIMARY KEY,
    owner      TEXT NOT NULL DEFAULT '',
    asn        TEXT NOT NULL DEFAULT '',
    as_name    TEXT NOT NULL DEFAULT '',
    location   TEXT NOT NULL DEFAULT '',
    ip_type    TEXT NOT NULL DEFAULT 'unknown',
    quality    TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);

CREATE TABLE node_aliases (
    alias_id   TEXT PRIMARY KEY,
    node_id    TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_aliases_node ON node_aliases(node_id);

CREATE TABLE runtime_settings (
    id                 INTEGER PRIMARY KEY,
    routing_mode       TEXT NOT NULL DEFAULT 'auto',
    force_country      TEXT NOT NULL DEFAULT '',
    routing_ip_type    TEXT NOT NULL DEFAULT 'all',
    connection_enabled INTEGER NOT NULL DEFAULT 1,
    fixed_node_id      TEXT
);
INSERT INTO runtime_settings (id) VALUES (1);

CREATE TABLE favorites (
    node_id TEXT PRIMARY KEY
);

CREATE TABLE node_blacklist (
    node_id    TEXT PRIMARY KEY,
    reason     TEXT NOT NULL DEFAULT '',
    marked_at  TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
CREATE INDEX idx_blacklist_expires ON node_blacklist(expires_at);

-- +goose Down
DROP TABLE node_blacklist;
DROP TABLE favorites;
DROP TABLE runtime_settings;
DROP TABLE node_aliases;
DROP TABLE ip_info_cache;
DROP TABLE proxy_nodes;
