-- name: InsertProbeResult :one
INSERT INTO probe_results (node_id, available, latency_ms, probed_at, result)
VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: ListProbesForNode :many
SELECT * FROM probe_results
WHERE node_id = ?
ORDER BY probed_at DESC
LIMIT ?;

-- name: DeleteOldProbes :exec
DELETE FROM probe_results WHERE probed_at < ?;
