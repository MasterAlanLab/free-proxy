-- name: GetIPInfo :one
SELECT * FROM ip_info_cache WHERE ip_address = ?;

-- name: UpsertIPInfo :exec
INSERT INTO ip_info_cache (ip_address, owner, asn, as_name, location, ip_type, quality, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(ip_address) DO UPDATE SET
    owner      = excluded.owner,
    asn        = excluded.asn,
    as_name    = excluded.as_name,
    location   = excluded.location,
    ip_type    = excluded.ip_type,
    quality    = excluded.quality,
    updated_at = excluded.updated_at;

-- name: AddNodeAlias :exec
INSERT INTO node_aliases (alias_id, node_id, created_at) VALUES (?, ?, ?)
ON CONFLICT(alias_id) DO NOTHING;

-- name: GetNodeAlias :one
SELECT * FROM node_aliases WHERE alias_id = ?;
