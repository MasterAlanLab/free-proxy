-- name: GetRuntimeSettings :one
SELECT * FROM runtime_settings WHERE id = 1;

-- name: UpdateRuntimeSettings :exec
UPDATE runtime_settings SET
    routing_mode       = ?,
    force_country      = ?,
    routing_ip_type    = ?,
    connection_enabled = ?,
    fixed_node_id      = ?
WHERE id = 1;

-- name: SetConnectionEnabled :exec
UPDATE runtime_settings SET connection_enabled = ? WHERE id = 1;

-- name: SetFixedNode :exec
UPDATE runtime_settings SET fixed_node_id = ? WHERE id = 1;

-- name: ListFavorites :many
SELECT node_id FROM favorites ORDER BY node_id;

-- name: IsFavorite :one
SELECT EXISTS(SELECT 1 FROM favorites WHERE node_id = ?);

-- name: AddFavorite :exec
INSERT INTO favorites (node_id) VALUES (?) ON CONFLICT(node_id) DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM favorites WHERE node_id = ?;

-- name: ListBlacklist :many
SELECT * FROM node_blacklist ORDER BY expires_at;

-- name: GetBlacklistEntry :one
SELECT * FROM node_blacklist WHERE node_id = ?;

-- name: UpsertBlacklist :exec
INSERT INTO node_blacklist (node_id, reason, marked_at, expires_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(node_id) DO UPDATE SET
    reason     = excluded.reason,
    marked_at  = excluded.marked_at,
    expires_at = excluded.expires_at;

-- name: DeleteBlacklist :exec
DELETE FROM node_blacklist WHERE node_id = ?;

-- name: DeleteExpiredBlacklist :exec
DELETE FROM node_blacklist WHERE expires_at <= ?;
