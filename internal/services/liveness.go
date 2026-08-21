package services

import (
	"context"
	"log/slog"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// LivenessService keeps the pool honest by deleting nodes rather than hiding
// them. It exists because the two questions "is this host still there" and "can
// we complete an OpenVPN handshake with it" need very different instruments: the
// handshake probe is slow and noisy (a node can fail several in a row and still
// be reachable), while a plain TCP dial to the node's own endpoint is fast,
// cheap, and a hard verdict on whether anything is listening.
//
// So the sweep dials every TCP node, counts consecutive failures, and hands that
// counter to the deletion rule. Nothing here decides usability — that stays with
// the OpenVPN probe and the node's status.
//
// These are constants rather than settings on purpose. Each was picked from a
// measurement against a real pool of ~1600 nodes, and the measurement is what
// justifies the number — splitting the two apart would leave someone tuning a
// knob with no way to see what it was traded against.
const (
	// A full sweep of ~1350 TCP nodes takes about 2.5 minutes at this
	// concurrency, so twice a day costs a few minutes of background dialling.
	// Concurrency stays low and the order is shuffled (see Sweep) to keep a
	// sweep looking like ordinary client traffic rather than a port scan.
	sweepInterval    = 12 * time.Hour
	sweepConcurrency = 20
	dialTimeout      = 5 * time.Second

	// Deleting needs both signals, and each threshold is set where the evidence
	// stops being ambiguous. Measured against live nodes: a host still on the
	// provider listing answered a dial 74% of the time whether its handshake
	// counter was 0 or above 5, while a host that had dropped off the listing
	// answered only 32% of the time. So absence from the listing is the
	// discriminating signal, and repeated dial failures confirm it.
	deleteAfterFailures = 3
	deleteAfterUnseen   = 24 * time.Hour

	// UDP nodes get no dial check — under a tenth of them answer ICMP, so a
	// ping verdict would delete firewalled-but-working nodes. They fall back to
	// the OpenVPN failure counter, which recovers often enough (43% of nodes
	// bounce back after 4 consecutive failures) that both bars must be higher.
	udpDeleteAfterFailures = 5
	udpDeleteAfterUnseen   = 72 * time.Hour
)

type LivenessService struct {
	nodes   *store.NodeRepository
	gateway *GatewayService

	// dial is swapped out in tests; it reports whether addr accepted a
	// connection within timeout.
	dial func(ctx context.Context, addr string, timeout time.Duration) bool

	mu sync.Mutex
}

// NewLivenessService constructs a LivenessService.
func NewLivenessService(nodes *store.NodeRepository, gateway *GatewayService) *LivenessService {
	return &LivenessService{nodes: nodes, gateway: gateway, dial: dialTCP}
}

func dialTCP(ctx context.Context, addr string, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Sweep dials the whole pool, records the outcome, and deletes nodes that both
// the reachability counter and the provider listing agree are gone.
func (s *LivenessService) Sweep(ctx context.Context) (domain.LivenessResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	targets, err := s.nodes.ListTCPLivenessTargets(ctx)
	if err != nil {
		return domain.LivenessResult{}, err
	}
	slog.Info("starting liveness sweep", "module", "liveness", "targets", len(targets))

	// Shuffling matters: the stored order groups nodes by country and provider
	// id, so sweeping it directly would walk one hoster's address range in
	// sequence. Randomised order plus low concurrency keeps a sweep looking like
	// ordinary client traffic spread over minutes.
	rand.Shuffle(len(targets), func(i, j int) { targets[i], targets[j] = targets[j], targets[i] })

	alive, dead := s.dialAll(ctx, targets)
	if ctx.Err() != nil {
		return domain.LivenessResult{}, ctx.Err()
	}

	rule := s.rule()
	if err := s.nodes.RecordLiveness(ctx, alive, dead, rule.KeepNodeID, time.Now().UTC()); err != nil {
		return domain.LivenessResult{}, err
	}

	deleted, err := s.nodes.DeleteDeadNodes(ctx, rule)
	if err != nil {
		return domain.LivenessResult{}, err
	}

	res := domain.LivenessResult{
		Checked: len(targets), Alive: len(alive), Dead: len(dead), Deleted: int(deleted),
	}
	slog.Info("liveness sweep complete", "module", "liveness",
		"checked", res.Checked, "alive", res.Alive, "dead", res.Dead, "deleted", res.Deleted)
	return res, nil
}

// dialAll probes every target under a fixed concurrency budget, returning the
// ids that answered and those that did not.
func (s *LivenessService) dialAll(ctx context.Context, targets []store.LivenessTarget) (alive, dead []string) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, sweepConcurrency)
	for _, t := range targets {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(t store.LivenessTarget) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ok := s.dial(ctx, net.JoinHostPort(t.RemoteHost, strconv.Itoa(t.RemotePort)), dialTimeout)
			mu.Lock()
			defer mu.Unlock()
			if ok {
				alive = append(alive, t.ID)
			} else {
				dead = append(dead, t.ID)
			}
		}(t)
	}
	wg.Wait()
	return alive, dead
}

// rule builds the deletion thresholds, pinning the active node so a sweep can
// never delete the node currently carrying traffic.
func (s *LivenessService) rule() store.DeadNodeRule {
	now := time.Now()
	rule := store.DeadNodeRule{
		TCPFailures:     deleteAfterFailures,
		TCPUnseenBefore: now.Add(-deleteAfterUnseen),
		UDPFailures:     udpDeleteAfterFailures,
		UDPUnseenBefore: now.Add(-udpDeleteAfterUnseen),
	}
	if s.gateway != nil {
		if active := s.gateway.Status().ActiveNodeID; active != nil {
			rule.KeepNodeID = *active
		}
	}
	return rule
}

// SweepJob is the JobFunc form of Sweep.
func (s *LivenessService) SweepJob(ctx context.Context) (map[string]any, error) {
	res, err := s.Sweep(ctx)
	if err != nil {
		return nil, err
	}
	return toMap(res)
}

// LivenessMonitor runs the sweep on an interval.
type LivenessMonitor struct {
	liveness *LivenessService
	State    MonitorState
}

// NewLivenessMonitor constructs a LivenessMonitor.
func NewLivenessMonitor(liveness *LivenessService) *LivenessMonitor {
	return &LivenessMonitor{liveness: liveness}
}

// firstSweepDelay keeps the sweep off the boot path — startup already has
// discovery, probing and tunnel setup competing for the network — while still
// getting a first reachability reading long before the full interval elapses.
const firstSweepDelay = 5 * time.Minute

// Run loops until ctx is cancelled.
func (m *LivenessMonitor) Run(ctx context.Context) {
	delay := firstSweepDelay
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		if _, err := m.liveness.Sweep(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			m.State.Heartbeat(false, err.Error())
			slog.Warn("liveness sweep failed", "module", "liveness", "err", err)
		} else {
			m.State.Heartbeat(true, "")
		}
		delay = sweepInterval
	}
}
