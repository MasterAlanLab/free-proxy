package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store/gen"
)

// ResolveAlias maps a possibly-stale node id to its canonical id.
func (r *NodeRepository) ResolveAlias(ctx context.Context, id string) string {
	alias, err := r.q.GetNodeAlias(ctx, id)
	if err != nil {
		return id
	}
	return alias.NodeID
}

// UpsertDiscovered upserts discovered nodes, preserving canonical ids and adding
// aliases when a node's generated id differs from the stored one. Returns count.
func (r *NodeRepository) UpsertDiscovered(ctx context.Context, nodes []domain.DiscoveredNode) (int, error) {
	for _, n := range nodes {
		identity := n.ProviderIdentity
		if identity == "" {
			identity = n.Provider + ":" + n.IPAddress
		}
		n.ProviderIdentity = identity
		existing, err := r.q.GetNodeByIdentity(ctx, gen.GetNodeByIdentityParams{Provider: n.Provider, ProviderIdentity: identity})
		if err == nil {
			if existing.ID != n.ID {
				_ = r.q.AddNodeAlias(ctx, gen.AddNodeAliasParams{AliasID: n.ID, NodeID: existing.ID, CreatedAt: tstr(time.Now())})
			}
			n.ID = existing.ID
		} else if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		if err := r.InsertDiscovered(ctx, n); err != nil {
			return 0, err
		}
	}
	return len(nodes), nil
}

// MarkProviderSnapshot records which nodes appeared in the latest successful
// discovery. VPNGate deliberately publishes only ~100 of its several thousand
// servers per request, rotating which ones, so absence from a response says
// almost nothing about whether a node still works — measured reachability for
// absent-but-recently-listed nodes stays well above half. source_present is
// therefore a display marker only: it answers "is this in the newest upstream
// list", and nothing selects nodes by it. Whether a node is usable is decided by
// status (OpenVPN probe) and whether it still exists at all by the liveness
// sweep, which deletes nodes rather than hiding them.
func (r *NodeRepository) MarkProviderSnapshot(ctx context.Context, provider string, identities []string) error {
	if len(identities) == 0 {
		_, err := r.db.ExecContext(ctx, "UPDATE proxy_nodes SET source_present = 0 WHERE provider = ?", provider)
		return err
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(identities)), ",")
	args := make([]any, 0, len(identities)+1)
	args = append(args, provider)
	for _, id := range identities {
		args = append(args, id)
	}
	query := "UPDATE proxy_nodes SET source_present = 0 WHERE provider = ? AND provider_identity NOT IN (" + placeholders + ")"
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// restoreStatusFromProbe rebuilds a node's status from the verdict its last
// OpenVPN handshake left behind. A cooldown is a timeout, not a verdict — but
// letting it expire used to write 'unavailable' unconditionally, so any healthy
// node that was ever blacklisted stayed demoted until some later probe cycle
// happened to re-test it, invisible to selection the whole time. Restoring
// without probing must not invent a verdict either, hence reading the old one.
const restoreStatusFromProbe = `CASE
		WHEN last_probed_at IS NULL THEN 'discovered'
		WHEN last_success_at IS NOT NULL AND last_success_at >= last_probed_at THEN 'ready'
		ELSE 'unavailable' END`

// ActiveBlacklistIDs clears expired entries (restoring node status) and returns
// the set of currently-blacklisted node ids.
func (r *NodeRepository) ActiveBlacklistIDs(ctx context.Context) (map[string]bool, error) {
	now := tstr(time.Now())
	expired, err := r.expiredBlacklistIDs(ctx, now)
	if err != nil {
		return nil, err
	}
	if len(expired) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(expired)), ",")
		args := make([]any, 0, len(expired))
		for _, id := range expired {
			args = append(args, id)
		}
		_, _ = r.db.ExecContext(ctx, "UPDATE proxy_nodes SET status = "+restoreStatusFromProbe+", cooldown_until = NULL WHERE id IN ("+ph+")", args...)
		_, _ = r.db.ExecContext(ctx, "DELETE FROM node_blacklist WHERE node_id IN ("+ph+")", args...)
	}
	rows, err := r.db.QueryContext(ctx, "SELECT node_id FROM node_blacklist WHERE expires_at > ?", now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (r *NodeRepository) expiredBlacklistIDs(ctx context.Context, now string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT node_id FROM node_blacklist WHERE expires_at <= ?", now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ClearExpiredBlacklist deletes expired blacklist rows and restores the nodes.
func (r *NodeRepository) ClearExpiredBlacklist(ctx context.Context) error {
	now := tstr(time.Now())
	expired, err := r.expiredBlacklistIDs(ctx, now)
	if err != nil {
		return err
	}
	if len(expired) == 0 {
		return nil
	}
	ph := strings.TrimSuffix(strings.Repeat("?,", len(expired)), ",")
	args := make([]any, 0, len(expired))
	for _, id := range expired {
		args = append(args, id)
	}
	_, _ = r.db.ExecContext(ctx, "UPDATE proxy_nodes SET status = "+restoreStatusFromProbe+", cooldown_until = NULL WHERE id IN ("+ph+")", args...)
	_, err = r.db.ExecContext(ctx, "DELETE FROM node_blacklist WHERE node_id IN ("+ph+")", args...)
	return err
}

// PurgeStaleNodes deletes nodes absent from the source since before the grace
// window, excluding favorites, blacklisted, and the fixed node.
func (r *NodeRepository) PurgeStaleNodes(ctx context.Context, grace time.Duration) (int64, error) {
	cutoff := tstr(time.Now().Add(-grace))
	res, err := r.db.ExecContext(ctx, `DELETE FROM proxy_nodes
		WHERE source_present = 0
		  AND last_seen_at IS NOT NULL
		  AND last_seen_at < ?
		  AND id NOT IN (SELECT node_id FROM favorites)
		  AND id NOT IN (SELECT node_id FROM node_blacklist)
		  AND id NOT IN (SELECT fixed_node_id FROM runtime_settings WHERE id = 1 AND fixed_node_id IS NOT NULL)`,
		cutoff)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// LivenessTarget is the minimal node view a liveness sweep needs to dial.
type LivenessTarget struct {
	ID         string
	RemoteHost string
	RemotePort int
}

// ListTCPLivenessTargets returns the dial target of every TCP node. UDP nodes
// are excluded because there is no cheap way to tell a live UDP endpoint from a
// dead one: ICMP is answered by under a tenth of them, so a ping-based verdict
// would delete nodes that are merely firewalled.
func (r *NodeRepository) ListTCPLivenessTargets(ctx context.Context) ([]LivenessTarget, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, remote_host, remote_port FROM proxy_nodes WHERE transport = 'tcp' AND remote_host != '' AND remote_port > 0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LivenessTarget
	for rows.Next() {
		var t LivenessTarget
		if err := rows.Scan(&t.ID, &t.RemoteHost, &t.RemotePort); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// RecordLiveness clears the failure counter for nodes that answered and bumps it
// for those that did not, in one transaction so a partial sweep cannot leave the
// pool half-updated.
func (r *NodeRepository) RecordLiveness(ctx context.Context, aliveIDs, deadIDs []string, keepNodeID string, now time.Time) error {
	if len(aliveIDs) == 0 && len(deadIDs) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := execInChunks(ctx, tx,
		"UPDATE proxy_nodes SET liveness_failures = 0, last_alive_at = ? WHERE id IN ", []any{tstr(now)}, aliveIDs); err != nil {
		return err
	}
	// Undo the demotion below for hosts that answer again. Without this the
	// sweep only ever moved nodes downward: a host that went away for one sweep
	// and came back stayed 'unavailable' forever, since clearing the counter
	// alone leaves it out of every selection path. Only rows whose last handshake
	// succeeded go back to 'ready' — a TCP dial is not evidence a tunnel builds,
	// so nodes that genuinely failed their probe stay where the probe put them.
	if err := execInChunks(ctx, tx,
		`UPDATE proxy_nodes SET status = 'ready' WHERE status = 'unavailable'
		 AND last_probed_at IS NOT NULL AND last_success_at IS NOT NULL
		 AND last_success_at >= last_probed_at AND id IN `, nil, aliveIDs); err != nil {
		return err
	}
	if err := execInChunks(ctx, tx,
		"UPDATE proxy_nodes SET liveness_failures = liveness_failures + 1 WHERE id IN ", nil, deadIDs); err != nil {
		return err
	}
	// A node whose own TCP endpoint refuses a connection cannot complete an
	// OpenVPN-over-TCP handshake either, so demote it now instead of leaving it
	// advertised as ready until the next probe cycle gets around to it. Only
	// ready rows are touched: statuses like cooldown carry state this sweep has
	// no business overwriting. keepNodeID is skipped so a sweep never edits the
	// row the gateway is actively using.
	if err := execInChunks(ctx, tx,
		"UPDATE proxy_nodes SET status = 'unavailable' WHERE status = 'ready' AND id != ? AND id IN ",
		[]any{keepNodeID}, deadIDs); err != nil {
		return err
	}
	return tx.Commit()
}

// execInChunks runs stmt once per batch of ids, appending an IN (...) list. The
// batches keep each statement under SQLite's bound-parameter ceiling.
func execInChunks(ctx context.Context, tx *sql.Tx, stmt string, lead []any, ids []string) error {
	const chunk = 400
	for start := 0; start < len(ids); start += chunk {
		end := start + chunk
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]
		args := make([]any, 0, len(lead)+len(batch))
		args = append(args, lead...)
		for _, id := range batch {
			args = append(args, id)
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		if _, err := tx.ExecContext(ctx, stmt+"("+placeholders+")", args...); err != nil {
			return err
		}
	}
	return nil
}

// DeadNodeRule describes when a node is considered gone for good. Both signals
// must agree before anything is deleted: the node has to have failed its own
// reachability check repeatedly *and* have dropped off the provider listing.
// Either signal alone is too weak — a node can fail several OpenVPN handshakes
// in a row and still be perfectly reachable, and a node can vanish from the
// provider listing purely because the rotation did not pick it this time.
type DeadNodeRule struct {
	TCPFailures     int
	TCPUnseenBefore time.Time
	// UDP nodes have no cheap reachability check, so they fall back to the
	// OpenVPN failure counter with a deliberately higher bar on both axes.
	UDPFailures     int
	UDPUnseenBefore time.Time
	// KeepNodeID is the node currently carrying traffic, never deleted.
	KeepNodeID string
}

// DeleteDeadNodes removes nodes both signals agree are gone, excluding the
// active node, favorites, blacklisted, and the fixed node.
func (r *NodeRepository) DeleteDeadNodes(ctx context.Context, rule DeadNodeRule) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM proxy_nodes
		WHERE (
		      (transport =  'tcp' AND liveness_failures    >= ? AND last_seen_at IS NOT NULL AND last_seen_at < ?)
		   OR (transport <> 'tcp' AND consecutive_failures >= ? AND last_seen_at IS NOT NULL AND last_seen_at < ?)
		)
		  AND id != ?
		  AND id NOT IN (SELECT node_id FROM favorites)
		  AND id NOT IN (SELECT node_id FROM node_blacklist)
		  AND id NOT IN (SELECT fixed_node_id FROM runtime_settings WHERE id = 1 AND fixed_node_id IS NOT NULL)`,
		rule.TCPFailures, tstr(rule.TCPUnseenBefore),
		rule.UDPFailures, tstr(rule.UDPUnseenBefore),
		rule.KeepNodeID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// Blacklist records a cooldown for a node with the given backoff.
func (r *NodeRepository) Blacklist(ctx context.Context, id, reason string, backoff time.Duration) error {
	id = r.ResolveAlias(ctx, id)
	now := time.Now()
	expires := now.Add(backoff)
	if err := r.q.UpsertBlacklist(ctx, gen.UpsertBlacklistParams{
		NodeID: id, Reason: reason, MarkedAt: tstr(now), ExpiresAt: tstr(expires),
	}); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx,
		"UPDATE proxy_nodes SET status = 'cooldown', cooldown_until = ? WHERE id = ?",
		tstr(expires), id)
	return err
}

// MarkProbing sets a node's status to probing.
func (r *NodeRepository) MarkProbing(ctx context.Context, id string) error {
	id = r.ResolveAlias(ctx, id)
	return r.SetStatus(ctx, id, domain.NodeProbing)
}

// MarkUnavailable marks a node unavailable and bumps failure counters.
func (r *NodeRepository) MarkUnavailable(ctx context.Context, id string) error {
	id = r.ResolveAlias(ctx, id)
	_, err := r.db.ExecContext(ctx,
		"UPDATE proxy_nodes SET status = 'unavailable', consecutive_failures = consecutive_failures + 1, failure_count = failure_count + 1 WHERE id = ?",
		id)
	return err
}

// UpdateProbeResult applies the status/counters transition after a probe.
func (r *NodeRepository) UpdateProbeResult(ctx context.Context, id string, available bool, latencyMS int, probedAt time.Time) error {
	id = r.ResolveAlias(ctx, id)
	ts := tstr(probedAt)
	if available {
		_, err := r.db.ExecContext(ctx,
			`UPDATE proxy_nodes SET status = 'ready', latency_ms = ?, last_probed_at = ?, last_success_at = ?,
			 consecutive_failures = 0, success_count = success_count + 1 WHERE id = ?`,
			latencyMS, ts, ts, id)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE proxy_nodes SET status = 'unavailable', latency_ms = ?, last_probed_at = ?,
		 consecutive_failures = consecutive_failures + 1, failure_count = failure_count + 1 WHERE id = ?`,
		latencyMS, ts, id)
	return err
}
