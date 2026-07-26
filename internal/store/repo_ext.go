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

// MarkProviderSnapshot flags nodes of a provider absent unless their identity is
// in the present set (the latest successful discovery).
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
		_, _ = r.db.ExecContext(ctx, "UPDATE proxy_nodes SET status = 'unavailable', cooldown_until = NULL WHERE id IN ("+ph+")", args...)
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
	_, _ = r.db.ExecContext(ctx, "UPDATE proxy_nodes SET status = 'unavailable', cooldown_until = NULL WHERE id IN ("+ph+")", args...)
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
