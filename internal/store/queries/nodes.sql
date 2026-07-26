-- name: GetNode :one
SELECT * FROM proxy_nodes WHERE id = ?;

-- name: GetNodeByIdentity :one
SELECT * FROM proxy_nodes WHERE provider = ? AND provider_identity = ?;

-- name: ListNodeIDs :many
SELECT id FROM proxy_nodes;

-- name: DeleteNode :exec
DELETE FROM proxy_nodes WHERE id = ?;

-- name: MarkAllNodesAbsent :exec
UPDATE proxy_nodes SET source_present = 0;

-- name: DeleteStaleAbsentNodes :exec
DELETE FROM proxy_nodes
WHERE source_present = 0
  AND (last_seen_at IS NULL OR last_seen_at < ?);

-- name: InsertDiscoveredNode :exec
INSERT INTO proxy_nodes (
    id, provider, provider_node_id, provider_identity, country, country_code,
    host_name, ip_address, remote_host, remote_port, transport,
    source_score, source_ping_ms, source_speed_bps, source_sessions,
    config_text, status, fetched_at, last_seen_at, source_present
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'discovered', ?, ?, 1
)
ON CONFLICT(provider, provider_identity) DO UPDATE SET
    provider_node_id = excluded.provider_node_id,
    country          = excluded.country,
    country_code     = excluded.country_code,
    host_name        = excluded.host_name,
    ip_address       = excluded.ip_address,
    remote_host      = excluded.remote_host,
    remote_port      = excluded.remote_port,
    transport        = excluded.transport,
    source_score     = excluded.source_score,
    source_ping_ms   = excluded.source_ping_ms,
    source_speed_bps = excluded.source_speed_bps,
    source_sessions  = excluded.source_sessions,
    config_text      = excluded.config_text,
    fetched_at       = excluded.fetched_at,
    last_seen_at     = excluded.last_seen_at,
    source_present   = 1;

-- name: UpdateNodeProbeOutcome :exec
UPDATE proxy_nodes SET
    status               = ?,
    latency_ms           = ?,
    consecutive_failures = ?,
    success_count        = ?,
    failure_count        = ?,
    last_probed_at       = ?,
    last_success_at      = ?,
    cooldown_until       = ?
WHERE id = ?;

-- name: UpdateNodeIPInfo :exec
UPDATE proxy_nodes SET
    ip_type            = ?,
    owner              = ?,
    asn                = ?,
    as_name            = ?,
    location           = ?,
    quality            = ?,
    ip_info_updated_at = ?
WHERE id = ?;

-- name: SetNodeStatus :exec
UPDATE proxy_nodes SET status = ? WHERE id = ?;
