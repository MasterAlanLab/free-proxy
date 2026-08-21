package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store/gen"
)

// ErrNotFound is returned when a lookup finds no matching row. It aliases the
// domain sentinel so callers can errors.Is against either.
var ErrNotFound = domain.ErrNotFound

// Repos aggregates the repository facades over the generated queries.
type Repos struct {
	DB       *sql.DB
	Q        *gen.Queries
	Nodes    *NodeRepository
	Settings *SettingsRepository
	Jobs     *JobRepository
	Probes   *ProbeResultRepository
	IPCache  *IPCacheRepository
	App      *AppSettingsRepository
}

// NewRepos constructs all repositories bound to db.
func NewRepos(db *sql.DB) *Repos {
	q := gen.New(db)
	return &Repos{
		DB:       db,
		Q:        q,
		Nodes:    &NodeRepository{db: db, q: q},
		Settings: &SettingsRepository{db: db, q: q},
		Jobs:     &JobRepository{db: db, q: q},
		Probes:   &ProbeResultRepository{db: db, q: q},
		IPCache:  &IPCacheRepository{db: db, q: q},
		App:      &AppSettingsRepository{db: db, root: db},
	}
}

// ---- conversion helpers -----------------------------------------------------

func tstr(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func tptr(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: tstr(*t), Valid: true}
}

func parseT(s string) time.Time {
	v, _ := time.Parse(time.RFC3339Nano, s)
	return v
}

func parseTPtr(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	v := parseT(ns.String)
	return &v
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func i2b(i int64) bool { return i != 0 }

func nsToStr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}

func strToNS(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// ---- node conversions -------------------------------------------------------

func nodeToRead(n gen.ProxyNode) domain.ProxyNodeRead {
	return domain.ProxyNodeRead{
		ID:                  n.ID,
		Provider:            n.Provider,
		ProviderNodeID:      n.ProviderNodeID,
		ProviderIdentity:    n.ProviderIdentity,
		Country:             n.Country,
		CountryCode:         n.CountryCode,
		HostName:            n.HostName,
		IPAddress:           n.IpAddress,
		RemoteHost:          n.RemoteHost,
		RemotePort:          int(n.RemotePort),
		Transport:           domain.TransportProtocol(n.Transport),
		IPType:              domain.IpType(n.IpType),
		Owner:               n.Owner,
		ASN:                 n.Asn,
		ASName:              n.AsName,
		Location:            n.Location,
		Quality:             n.Quality,
		Status:              domain.NodeStatus(n.Status),
		SourceScore:         int(n.SourceScore),
		SourcePingMS:        int(n.SourcePingMs),
		SourceSpeedBPS:      n.SourceSpeedBps,
		SourceSessions:      int(n.SourceSessions),
		LatencyMS:           int(n.LatencyMs),
		ConsecutiveFailures: int(n.ConsecutiveFailures),
		SuccessCount:        int(n.SuccessCount),
		FailureCount:        int(n.FailureCount),
		FetchedAt:           parseT(n.FetchedAt),
		LastProbedAt:        parseTPtr(n.LastProbedAt),
		LastSuccessAt:       parseTPtr(n.LastSuccessAt),
		IPInfoUpdatedAt:     parseTPtr(n.IpInfoUpdatedAt),
		CooldownUntil:       parseTPtr(n.CooldownUntil),
		LastSeenAt:          parseTPtr(n.LastSeenAt),
		SourcePresent:       i2b(n.SourcePresent),
	}
}

const nodeColumns = `id, provider, provider_node_id, provider_identity, country, country_code,
	host_name, ip_address, remote_host, remote_port, transport, ip_type, owner, asn, as_name,
	location, quality, status, source_score, source_ping_ms, source_speed_bps, source_sessions,
	latency_ms, consecutive_failures, success_count, failure_count, config_text, fetched_at,
	last_probed_at, last_success_at, ip_info_updated_at, cooldown_until, last_seen_at, source_present`

type rowScanner interface{ Scan(dest ...any) error }

func scanNode(s rowScanner) (gen.ProxyNode, error) {
	var n gen.ProxyNode
	err := s.Scan(
		&n.ID, &n.Provider, &n.ProviderNodeID, &n.ProviderIdentity, &n.Country, &n.CountryCode,
		&n.HostName, &n.IpAddress, &n.RemoteHost, &n.RemotePort, &n.Transport, &n.IpType, &n.Owner,
		&n.Asn, &n.AsName, &n.Location, &n.Quality, &n.Status, &n.SourceScore, &n.SourcePingMs,
		&n.SourceSpeedBps, &n.SourceSessions, &n.LatencyMs, &n.ConsecutiveFailures, &n.SuccessCount,
		&n.FailureCount, &n.ConfigText, &n.FetchedAt, &n.LastProbedAt, &n.LastSuccessAt,
		&n.IpInfoUpdatedAt, &n.CooldownUntil, &n.LastSeenAt, &n.SourcePresent,
	)
	return n, err
}

// ---- NodeRepository ---------------------------------------------------------

type NodeRepository struct {
	db *sql.DB
	q  *gen.Queries
}

// NodeFilter captures the optional list filters exposed by the API.
type NodeFilter struct {
	IPType       string
	Status       string
	Country      string
	Search       string
	FavoriteOnly bool
	// ListedOnly narrows to nodes in the provider's newest published list. It is
	// a diagnostic view, not a usability filter — see MarkProviderSnapshot.
	ListedOnly bool
	// ReachableOnly drops nodes the last liveness sweep could not reach. A host
	// that refuses a TCP connection cannot complete an OpenVPN-over-TCP
	// handshake either, so this is how the cheap check protects the expensive
	// one from spending its whole budget on hosts already known to be gone.
	ReachableOnly bool
}

func (f NodeFilter) where() (string, []any) {
	var clauses []string
	var args []any
	if f.IPType != "" {
		clauses = append(clauses, "ip_type = ?")
		args = append(args, f.IPType)
	}
	if f.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, f.Status)
	}
	if f.Country != "" {
		clauses = append(clauses, "(country = ? OR country_code = ?)")
		args = append(args, f.Country, f.Country)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		clauses = append(clauses,
			"(ip_address LIKE ? OR host_name LIKE ? OR country LIKE ? OR remote_host LIKE ? OR provider_identity LIKE ?)")
		args = append(args, like, like, like, like, like)
	}
	if f.FavoriteOnly {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM favorites WHERE favorites.node_id = proxy_nodes.id)")
	}
	if f.ListedOnly {
		clauses = append(clauses, "source_present = 1")
	}
	if f.ReachableOnly {
		clauses = append(clauses, "liveness_failures = 0")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// ListNodes returns a page of nodes matching the filter.
func (r *NodeRepository) ListNodes(ctx context.Context, f NodeFilter, limit, offset int) ([]domain.ProxyNodeRead, error) {
	where, args := f.where()
	query := "SELECT " + nodeColumns + " FROM proxy_nodes" + where +
		" ORDER BY CASE WHEN latency_ms > 0 THEN latency_ms ELSE 999999 END ASC, source_score DESC, id ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.ProxyNodeRead, 0, limit)
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, nodeToRead(n))
	}
	return out, rows.Err()
}

// CountNodes counts nodes matching the filter.
func (r *NodeRepository) CountNodes(ctx context.Context, f NodeFilter) (int64, error) {
	where, args := f.where()
	var total int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM proxy_nodes"+where, args...).Scan(&total)
	return total, err
}

// Get returns one node by id (resolving aliases).
func (r *NodeRepository) Get(ctx context.Context, id string) (domain.ProxyNodeRead, error) {
	id = r.ResolveAlias(ctx, id)
	n, err := r.q.GetNode(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProxyNodeRead{}, ErrNotFound
	}
	if err != nil {
		return domain.ProxyNodeRead{}, err
	}
	return nodeToRead(n), nil
}

// GetTarget returns the activation target (config + remote) for a node (resolving aliases).
func (r *NodeRepository) GetTarget(ctx context.Context, id string) (domain.ProxyNodeTarget, error) {
	id = r.ResolveAlias(ctx, id)
	n, err := r.q.GetNode(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProxyNodeTarget{}, ErrNotFound
	}
	if err != nil {
		return domain.ProxyNodeTarget{}, err
	}
	return domain.ProxyNodeTarget{
		ID:           n.ID,
		IPAddress:    n.IpAddress,
		RemoteHost:   n.RemoteHost,
		RemotePort:   int(n.RemotePort),
		SourcePingMS: int(n.SourcePingMs),
		ConfigText:   n.ConfigText,
	}, nil
}

// InsertDiscovered upserts a freshly discovered node.
func (r *NodeRepository) InsertDiscovered(ctx context.Context, n domain.DiscoveredNode) error {
	return r.q.InsertDiscoveredNode(ctx, gen.InsertDiscoveredNodeParams{
		ID:               n.ID,
		Provider:         n.Provider,
		ProviderNodeID:   n.ProviderNodeID,
		ProviderIdentity: n.ProviderIdentity,
		Country:          n.Country,
		CountryCode:      n.CountryCode,
		HostName:         n.HostName,
		IpAddress:        n.IPAddress,
		RemoteHost:       n.RemoteHost,
		RemotePort:       int64(n.RemotePort),
		Transport:        string(n.Transport),
		SourceScore:      int64(n.SourceScore),
		SourcePingMs:     int64(n.SourcePingMS),
		SourceSpeedBps:   n.SourceSpeedBPS,
		SourceSessions:   int64(n.SourceSessions),
		ConfigText:       n.ConfigText,
		FetchedAt:        tstr(n.FetchedAt),
		LastSeenAt:       sql.NullString{String: tstr(n.FetchedAt), Valid: true},
	})
}

// MarkAllAbsent flags every node as not present in the latest source snapshot.
func (r *NodeRepository) MarkAllAbsent(ctx context.Context) error {
	return r.q.MarkAllNodesAbsent(ctx)
}

// DeleteStaleAbsent removes nodes absent from the source since before cutoff.
func (r *NodeRepository) DeleteStaleAbsent(ctx context.Context, cutoff time.Time) error {
	return r.q.DeleteStaleAbsentNodes(ctx, sql.NullString{String: tstr(cutoff), Valid: true})
}

// ProbeOutcome carries the mutable fields updated after a probe.
type ProbeOutcome struct {
	Status              domain.NodeStatus
	LatencyMS           int
	ConsecutiveFailures int
	SuccessCount        int
	FailureCount        int
	LastProbedAt        *time.Time
	LastSuccessAt       *time.Time
	CooldownUntil       *time.Time
}

// UpdateProbeOutcome persists the result of a probe against a node.
func (r *NodeRepository) UpdateProbeOutcome(ctx context.Context, id string, o ProbeOutcome) error {
	return r.q.UpdateNodeProbeOutcome(ctx, gen.UpdateNodeProbeOutcomeParams{
		Status:              string(o.Status),
		LatencyMs:           int64(o.LatencyMS),
		ConsecutiveFailures: int64(o.ConsecutiveFailures),
		SuccessCount:        int64(o.SuccessCount),
		FailureCount:        int64(o.FailureCount),
		LastProbedAt:        tptr(o.LastProbedAt),
		LastSuccessAt:       tptr(o.LastSuccessAt),
		CooldownUntil:       tptr(o.CooldownUntil),
		ID:                  id,
	})
}

// UpdateIPInfo persists IP classification for a node.
func (r *NodeRepository) UpdateIPInfo(ctx context.Context, id string, info domain.IpInfo, at time.Time) error {
	return r.q.UpdateNodeIPInfo(ctx, gen.UpdateNodeIPInfoParams{
		IpType:          string(info.IPType),
		Owner:           info.Owner,
		Asn:             info.ASN,
		AsName:          info.ASName,
		Location:        info.Location,
		Quality:         info.Quality,
		IpInfoUpdatedAt: sql.NullString{String: tstr(at), Valid: true},
		ID:              id,
	})
}

// SetStatus updates only a node's status.
func (r *NodeRepository) SetStatus(ctx context.Context, id string, status domain.NodeStatus) error {
	return r.q.SetNodeStatus(ctx, gen.SetNodeStatusParams{Status: string(status), ID: id})
}

// Delete removes a node.
func (r *NodeRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteNode(ctx, id)
}

// Statistics counts the whole pool. Every retained row is a node the liveness
// sweep has not disproved, so there is no longer a subset to exclude: what the
// dashboard counts and what the list shows are the same rows.
func (r *NodeRepository) Statistics(ctx context.Context) (domain.PoolStatistics, error) {
	var s domain.PoolStatistics
	row := r.db.QueryRowContext(ctx, `SELECT
		COUNT(*),
		SUM(CASE WHEN status='ready' THEN 1 ELSE 0 END),
		SUM(CASE WHEN status='discovered' THEN 1 ELSE 0 END),
		SUM(CASE WHEN status='unavailable' THEN 1 ELSE 0 END),
		SUM(CASE WHEN status='cooldown' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ip_type='residential' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ip_type='mobile' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ip_type='hosting' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ip_type='unknown' THEN 1 ELSE 0 END),
		COUNT(DISTINCT CASE WHEN country != '' THEN country END)
		FROM proxy_nodes`)
	var ready, disc, unavail, cool, res, mob, host, unk, countries sql.NullInt64
	var total int64
	if err := row.Scan(&total, &ready, &disc, &unavail, &cool, &res, &mob, &host, &unk, &countries); err != nil {
		return s, err
	}
	s.Total = total
	s.Ready = ready.Int64
	s.Discovered = disc.Int64
	s.Unavailable = unavail.Int64
	s.Cooldown = cool.Int64
	s.Residential = res.Int64
	s.Mobile = mob.Int64
	s.Hosting = host.Int64
	s.Unknown = unk.Int64
	s.Countries = countries.Int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM favorites").Scan(&s.Favorites); err != nil {
		return s, err
	}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM node_blacklist").Scan(&s.Blacklisted); err != nil {
		return s, err
	}
	return s, nil
}

// ---- SettingsRepository -----------------------------------------------------

type SettingsRepository struct {
	db *sql.DB
	q  *gen.Queries
}

// Get assembles the current proxy settings including favorites.
func (r *SettingsRepository) Get(ctx context.Context) (domain.ProxySettings, error) {
	rs, err := r.q.GetRuntimeSettings(ctx)
	if err != nil {
		return domain.ProxySettings{}, err
	}
	favs, err := r.q.ListFavorites(ctx)
	if err != nil {
		return domain.ProxySettings{}, err
	}
	return domain.ProxySettings{
		RoutingMode:       domain.ProxyPolicyMode(rs.RoutingMode),
		ForceCountry:      rs.ForceCountry,
		RoutingIPType:     domain.RoutingIpType(rs.RoutingIpType),
		ConnectionEnabled: i2b(rs.ConnectionEnabled),
		FixedNodeID:       nsToStr(rs.FixedNodeID),
		FavoriteNodeIDs:   favs,
	}, nil
}

// Update applies a settings update (favorites unchanged).
func (r *SettingsRepository) Update(ctx context.Context, u domain.ProxySettingsUpdate) error {
	return r.q.UpdateRuntimeSettings(ctx, gen.UpdateRuntimeSettingsParams{
		RoutingMode:       string(u.RoutingMode),
		ForceCountry:      u.ForceCountry,
		RoutingIpType:     string(u.RoutingIPType),
		ConnectionEnabled: b2i(u.ConnectionEnabled),
		FixedNodeID:       strToNS(u.FixedNodeID),
	})
}

// SetConnectionEnabled toggles the master connection switch.
func (r *SettingsRepository) SetConnectionEnabled(ctx context.Context, enabled bool) error {
	return r.q.SetConnectionEnabled(ctx, b2i(enabled))
}

// SetFixedNode records the pinned node id (nil clears it).
func (r *SettingsRepository) SetFixedNode(ctx context.Context, id *string) error {
	return r.q.SetFixedNode(ctx, strToNS(id))
}

// ToggleFavorite flips membership and returns the updated favorite list.
func (r *SettingsRepository) ToggleFavorite(ctx context.Context, nodeID string) ([]string, error) {
	exists, err := r.q.IsFavorite(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if exists {
		err = r.q.RemoveFavorite(ctx, nodeID)
	} else {
		err = r.q.AddFavorite(ctx, nodeID)
	}
	if err != nil {
		return nil, err
	}
	return r.q.ListFavorites(ctx)
}

// BlacklistEntry mirrors a node_blacklist row in domain terms.
type BlacklistEntry struct {
	NodeID    string
	Reason    string
	MarkedAt  time.Time
	ExpiresAt time.Time
}

// Blacklist marks a node unavailable until expiresAt.
func (r *SettingsRepository) Blacklist(ctx context.Context, nodeID, reason string, markedAt, expiresAt time.Time) error {
	return r.q.UpsertBlacklist(ctx, gen.UpsertBlacklistParams{
		NodeID:    nodeID,
		Reason:    reason,
		MarkedAt:  tstr(markedAt),
		ExpiresAt: tstr(expiresAt),
	})
}

// RemoveBlacklist clears a blacklist entry.
func (r *SettingsRepository) RemoveBlacklist(ctx context.Context, nodeID string) error {
	return r.q.DeleteBlacklist(ctx, nodeID)
}

// PurgeExpiredBlacklist deletes entries whose cooldown has elapsed.
func (r *SettingsRepository) PurgeExpiredBlacklist(ctx context.Context, now time.Time) error {
	return r.q.DeleteExpiredBlacklist(ctx, tstr(now))
}

// ListBlacklist returns all current blacklist entries.
func (r *SettingsRepository) ListBlacklist(ctx context.Context) ([]BlacklistEntry, error) {
	rows, err := r.q.ListBlacklist(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]BlacklistEntry, 0, len(rows))
	for _, b := range rows {
		out = append(out, BlacklistEntry{
			NodeID:    b.NodeID,
			Reason:    b.Reason,
			MarkedAt:  parseT(b.MarkedAt),
			ExpiresAt: parseT(b.ExpiresAt),
		})
	}
	return out, nil
}

// ---- JobRepository ----------------------------------------------------------

type JobRepository struct {
	db *sql.DB
	q  *gen.Queries
}

func (r *JobRepository) Create(ctx context.Context, id, name string, createdAt time.Time) error {
	return r.q.CreateJob(ctx, gen.CreateJobParams{
		ID: id, Name: name, Status: string(domain.JobPending), CreatedAt: tstr(createdAt),
	})
}

func (r *JobRepository) MarkRunning(ctx context.Context, id string, at time.Time) error {
	return r.q.MarkJobRunning(ctx, gen.MarkJobRunningParams{
		StartedAt: sql.NullString{String: tstr(at), Valid: true}, ID: id,
	})
}

func (r *JobRepository) MarkSucceeded(ctx context.Context, id string, at time.Time, result map[string]any) error {
	var payload sql.NullString
	if result != nil {
		if data, err := json.Marshal(result); err == nil {
			payload = sql.NullString{String: string(data), Valid: true}
		}
	}
	return r.q.MarkJobSucceeded(ctx, gen.MarkJobSucceededParams{
		FinishedAt: sql.NullString{String: tstr(at), Valid: true}, Result: payload, ID: id,
	})
}

func (r *JobRepository) MarkFailed(ctx context.Context, id string, at time.Time, errMsg string) error {
	return r.q.MarkJobFailed(ctx, gen.MarkJobFailedParams{
		FinishedAt: sql.NullString{String: tstr(at), Valid: true},
		Error:      sql.NullString{String: errMsg, Valid: true},
		ID:         id,
	})
}

func (r *JobRepository) CancelUnfinished(ctx context.Context, at time.Time) error {
	return r.q.CancelUnfinishedJobs(ctx, sql.NullString{String: tstr(at), Valid: true})
}

func (r *JobRepository) Get(ctx context.Context, id string) (domain.JobRead, error) {
	j, err := r.q.GetJob(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.JobRead{}, ErrNotFound
	}
	if err != nil {
		return domain.JobRead{}, err
	}
	out := domain.JobRead{
		ID:         j.ID,
		Name:       j.Name,
		Status:     domain.JobStatus(j.Status),
		CreatedAt:  parseT(j.CreatedAt),
		StartedAt:  parseTPtr(j.StartedAt),
		FinishedAt: parseTPtr(j.FinishedAt),
		Error:      nsToStr(j.Error),
	}
	if j.Result.Valid && j.Result.String != "" {
		_ = json.Unmarshal([]byte(j.Result.String), &out.Result)
	}
	return out, nil
}

// ---- ProbeResultRepository --------------------------------------------------

type ProbeResultRepository struct {
	db *sql.DB
	q  *gen.Queries
}

func (r *ProbeResultRepository) Insert(ctx context.Context, res domain.ProbeResult) (int64, error) {
	payload, err := json.Marshal(res)
	if err != nil {
		return 0, err
	}
	return r.q.InsertProbeResult(ctx, gen.InsertProbeResultParams{
		NodeID:    res.NodeID,
		Available: b2i(res.Available),
		LatencyMs: int64(res.LatencyMS),
		ProbedAt:  tstr(res.ProbedAt),
		Result:    string(payload),
	})
}

func (r *ProbeResultRepository) ListForNode(ctx context.Context, nodeID string, limit int) ([]domain.ProbeHistoryRead, error) {
	rows, err := r.q.ListProbesForNode(ctx, gen.ListProbesForNodeParams{NodeID: nodeID, Limit: int64(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.ProbeHistoryRead, 0, len(rows))
	for _, p := range rows {
		item := domain.ProbeHistoryRead{
			ID:        p.ID,
			NodeID:    p.NodeID,
			Available: i2b(p.Available),
			LatencyMS: int(p.LatencyMs),
			ProbedAt:  parseT(p.ProbedAt),
		}
		if p.Result != "" {
			_ = json.Unmarshal([]byte(p.Result), &item.Result)
		}
		out = append(out, item)
	}
	return out, nil
}

// ---- IPCacheRepository ------------------------------------------------------

type IPCacheRepository struct {
	db *sql.DB
	q  *gen.Queries
}

func (r *IPCacheRepository) Get(ctx context.Context, ip string) (domain.IpInfo, time.Time, error) {
	row, err := r.q.GetIPInfo(ctx, ip)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.IpInfo{}, time.Time{}, ErrNotFound
	}
	if err != nil {
		return domain.IpInfo{}, time.Time{}, err
	}
	return domain.IpInfo{
		IPAddress: row.IpAddress,
		Owner:     row.Owner,
		ASN:       row.Asn,
		ASName:    row.AsName,
		Location:  row.Location,
		IPType:    domain.IpType(row.IpType),
		Quality:   row.Quality,
	}, parseT(row.UpdatedAt), nil
}

func (r *IPCacheRepository) Upsert(ctx context.Context, info domain.IpInfo, at time.Time) error {
	return r.q.UpsertIPInfo(ctx, gen.UpsertIPInfoParams{
		IpAddress: info.IPAddress,
		Owner:     info.Owner,
		Asn:       info.ASN,
		AsName:    info.ASName,
		Location:  info.Location,
		IpType:    string(info.IPType),
		Quality:   info.Quality,
		UpdatedAt: tstr(at),
	})
}
