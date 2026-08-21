-- +goose Up
-- Liveness bookkeeping is deliberately separate from probe bookkeeping.
-- consecutive_failures counts OpenVPN handshake failures, which are dominated by
-- transient causes (congestion, probe contention, a 15s timeout) and therefore
-- predict host death poorly. liveness_failures counts failed TCP dials to the
-- node's own remote endpoint, which is a hard host-level signal and is what the
-- deletion rule keys off.
ALTER TABLE proxy_nodes ADD COLUMN liveness_failures INTEGER NOT NULL DEFAULT 0;
ALTER TABLE proxy_nodes ADD COLUMN last_alive_at TEXT;
CREATE INDEX idx_nodes_liveness ON proxy_nodes(liveness_failures, last_seen_at);

-- +goose Down
DROP INDEX idx_nodes_liveness;
ALTER TABLE proxy_nodes DROP COLUMN last_alive_at;
ALTER TABLE proxy_nodes DROP COLUMN liveness_failures;
