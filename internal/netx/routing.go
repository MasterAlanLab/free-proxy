package netx

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/masteralanlab/free-proxy/internal/naming"
)

// PolicyRouter installs the policy route/rule that forces marked traffic through
// the tunnel and relaxes rp_filter, restoring prior values on cleanup. Linux only.
//
// The routing table id is a global namespace shared with every other tool on
// the host, so this type never issues a blanket `ip rule del table N` /
// `ip route flush table N`: it removes only entries it can attribute to one of
// our own devices. An earlier version flushed unconditionally and would wipe a
// neighbouring tool's routes whenever the table ids happened to collide.
type PolicyRouter struct {
	runner        CommandRunner
	table         int
	iface         string
	devicePrefix  string
	setupRetries  int
	retryInterval time.Duration
	strictRPF     bool

	// supportedOverride lets tests exercise the Linux paths on any host.
	supportedOverride *bool

	rpFilterOriginal map[string]string
}

// PolicyRouterConfig carries the settings a PolicyRouter needs.
type PolicyRouterConfig struct {
	Table         int
	Interface     string
	DevicePrefix  string
	SetupRetries  int
	RetryInterval time.Duration
	StrictRPF     bool
}

// NewPolicyRouter constructs a PolicyRouter.
func NewPolicyRouter(runner CommandRunner, cfg PolicyRouterConfig) *PolicyRouter {
	if runner == nil {
		runner = SystemCommandRunner{}
	}
	if cfg.SetupRetries < 1 {
		cfg.SetupRetries = 1
	}
	if cfg.DevicePrefix == "" {
		cfg.DevicePrefix = naming.DevicePrefix
	}
	return &PolicyRouter{
		runner:           runner,
		table:            cfg.Table,
		iface:            cfg.Interface,
		devicePrefix:     cfg.DevicePrefix,
		setupRetries:     cfg.SetupRetries,
		retryInterval:    cfg.RetryInterval,
		strictRPF:        cfg.StrictRPF,
		rpFilterOriginal: map[string]string{},
	}
}

// Supported reports whether policy routing is available on this OS.
func (r *PolicyRouter) Supported() bool {
	if r.supportedOverride != nil {
		return *r.supportedOverride
	}
	return runtime.GOOS == "linux"
}

// Table returns the policy routing table currently in use.
func (r *PolicyRouter) Table() int { return r.table }

// owns reports whether a device is one this project creates. It is the single
// gate on every destructive routing operation.
func (r *PolicyRouter) owns(device string) bool {
	if device == "" {
		return false
	}
	if device == r.iface {
		return true
	}
	return naming.HasDevicePrefix(device, r.devicePrefix)
}

// Setup installs the route/rule with retries, cleaning up between attempts.
func (r *PolicyRouter) Setup(ctx context.Context, iface string) error {
	if !r.Supported() {
		return fmt.Errorf("policy routing is only supported on Linux")
	}
	device := iface
	if device == "" {
		device = r.iface
	}
	table := strconv.Itoa(r.table)
	var lastErr error
	for attempt := 1; attempt <= r.setupRetries; attempt++ {
		if err := r.setupOnce(ctx, device, table); err != nil {
			lastErr = err
			_ = r.Cleanup(ctx)
			if attempt < r.setupRetries && r.retryInterval > 0 {
				time.Sleep(r.retryInterval)
			}
			continue
		}
		return nil
	}
	return lastErr
}

func (r *PolicyRouter) setupOnce(ctx context.Context, device, table string) error {
	_ = r.Cleanup(ctx)

	routeRes, err := r.runner.Run(ctx, []string{"ip", "route", "add", "default", "dev", device, "table", table}, 5*time.Second)
	if err != nil {
		return err
	}
	if routeRes.ReturnCode != 0 {
		return fmt.Errorf("unable to add policy route for %s: %s", device, strings.TrimSpace(routeRes.Stderr))
	}
	ruleRes, err := r.runner.Run(ctx, []string{"ip", "rule", "add", "oif", device, "table", table}, 5*time.Second)
	if err != nil {
		return err
	}
	if ruleRes.ReturnCode != 0 {
		_ = r.Cleanup(ctx)
		return fmt.Errorf("unable to add policy rule for %s: %s", device, strings.TrimSpace(ruleRes.Stderr))
	}
	// Loosen reverse-path filtering on our tunnel device only.
	//
	// Return traffic arrives on the tunnel while the route sending traffic out
	// of it lives in our policy table, so a strict reverse-path lookup (which
	// consults the main table) would drop it. Loose mode on that device fixes
	// it.
	//
	// Scoped deliberately. The kernel uses max(conf.all, conf.<iface>) for an
	// interface, so setting the device to 2 already yields 2 whatever `all`
	// holds — while setting conf.all=2 (as this did previously) forces *every*
	// interface on the host to loose mode, silently disabling anti-spoofing for
	// programs that never asked us to. conf.default is likewise pointless here:
	// it only templates interfaces created later, and our device already exists
	// by the time Setup runs.
	key := "net.ipv4.conf." + device + ".rp_filter"
	if read, err := r.runner.Run(ctx, []string{"sysctl", "-n", key}, 5*time.Second); err == nil && read.ReturnCode == 0 {
		r.rpFilterOriginal[device] = strings.TrimSpace(read.Stdout)
	}
	set, err := r.runner.Run(ctx, []string{"sysctl", "-w", key + "=2"}, 5*time.Second)
	if err != nil || set.ReturnCode != 0 {
		msg := fmt.Sprintf("unable to configure rp_filter for %s", device)
		if r.strictRPF {
			return fmt.Errorf("%s", msg)
		}
		slog.Warn(msg, "module", "netx")
	}
	return nil
}

// Cleanup removes the rules and routes this project installed and restores
// rp_filter values. Entries in the table that point at a device we do not own
// are left strictly untouched.
func (r *PolicyRouter) Cleanup(ctx context.Context) error {
	if !r.Supported() {
		return nil
	}
	table := strconv.Itoa(r.table)

	for _, rule := range r.ourRules(ctx) {
		_, _ = r.runner.Run(ctx, []string{"ip", "rule", "del", "oif", rule.OIF, "table", table}, 5*time.Second)
	}

	ours, foreign := r.tableRoutes(ctx)
	switch {
	case len(foreign) == 0:
		// Nothing but our own routes (or none at all): a flush is safe and also
		// clears any entry we failed to parse.
		_, _ = r.runner.Run(ctx, []string{"ip", "route", "flush", "table", table}, 5*time.Second)
	default:
		for _, route := range ours {
			_, _ = r.runner.Run(ctx, []string{"ip", "route", "del", route.Destination, "dev", route.Device, "table", table}, 5*time.Second)
		}
		slog.Warn("policy routing table also holds entries from another program; left them in place",
			"module", "netx", "table", r.table, "foreign_routes", len(foreign))
	}

	for target, value := range r.rpFilterOriginal {
		_, _ = r.runner.Run(ctx, []string{"sysctl", "-w", "net.ipv4.conf." + target + ".rp_filter=" + value}, 5*time.Second)
	}
	r.rpFilterOriginal = map[string]string{}
	return nil
}

// TableConflict reports the number of routes in our table that belong to some
// other program. A non-zero count means the table id is shared and the operator
// should move one side out of the way.
func (r *PolicyRouter) TableConflict(ctx context.Context) int {
	if !r.Supported() {
		return 0
	}
	_, foreign := r.tableRoutes(ctx)
	return len(foreign)
}

func (r *PolicyRouter) ourRules(ctx context.Context) []PolicyRule {
	res, err := r.runner.Run(ctx, []string{"ip", "rule", "show"}, 5*time.Second)
	if err != nil || res.ReturnCode != 0 {
		return nil
	}
	var out []PolicyRule
	for _, rule := range ParsePolicyRules(res.Stdout) {
		if !rule.MatchesTable(r.table) || !r.owns(rule.OIF) {
			continue
		}
		out = append(out, rule)
	}
	return out
}

func (r *PolicyRouter) tableRoutes(ctx context.Context) (ours, foreign []TableRoute) {
	res, err := r.runner.Run(ctx, []string{"ip", "route", "show", "table", strconv.Itoa(r.table)}, 5*time.Second)
	if err != nil || res.ReturnCode != 0 {
		return nil, nil
	}
	for _, route := range ParseTableRoutes(res.Stdout) {
		if r.owns(route.Device) {
			ours = append(ours, route)
		} else {
			foreign = append(foreign, route)
		}
	}
	return ours, foreign
}

// CleanupOrphanedRules removes policy rules that select a device which does not
// exist on this host, plus the routes those rules' tables hold for that same
// device. Returns a description of what it removed.
//
// This exists for one-time cleanup of the identifiers earlier releases claimed
// (tun0 / table 100). Attribution there is impossible after the fact — we
// cannot tell our own leftover tun0 rule from a neighbour's — so the guard is
// liveness instead: a rule whose device is absent cannot be steering anyone's
// traffic today, and removing it is therefore safe. Leaving it, on the other
// hand, is not: the day something else creates a device by that name, the stale
// rule starts diverting *its* traffic into an empty table.
func CleanupOrphanedRules(ctx context.Context, runner CommandRunner, device string) []string {
	if runtime.GOOS != "linux" || device == "" {
		return nil
	}
	if DeviceExists(device) {
		return nil // live device: not an orphan, and not ours to judge
	}
	if runner == nil {
		runner = SystemCommandRunner{}
	}
	res, err := runner.Run(ctx, []string{"ip", "rule", "show"}, 5*time.Second)
	if err != nil || res.ReturnCode != 0 {
		return nil
	}

	var removed []string
	seenTables := map[string]bool{}
	for _, rule := range ParsePolicyRules(res.Stdout) {
		if rule.OIF != device || seenTables[rule.Table] {
			continue
		}
		seenTables[rule.Table] = true
		del, err := runner.Run(ctx, []string{"ip", "rule", "del", "oif", device, "table", rule.Table}, 5*time.Second)
		if err != nil || del.ReturnCode != 0 {
			continue
		}
		removed = append(removed, fmt.Sprintf("rule oif %s table %s", device, rule.Table))

		// The table may also still hold that device's routes. Same liveness
		// argument, same narrow scope: only routes naming the absent device.
		routes, err := runner.Run(ctx, []string{"ip", "route", "show", "table", rule.Table}, 5*time.Second)
		if err != nil || routes.ReturnCode != 0 {
			continue
		}
		for _, route := range ParseTableRoutes(routes.Stdout) {
			if route.Device != device {
				continue
			}
			if _, err := runner.Run(ctx, []string{"ip", "route", "del", route.Destination, "dev", device, "table", rule.Table}, 5*time.Second); err == nil {
				removed = append(removed, fmt.Sprintf("route %s dev %s table %s", route.Destination, device, rule.Table))
			}
		}
	}
	return removed
}

// PolicyRule is one parsed line of `ip rule show`.
type PolicyRule struct {
	Priority string
	OIF      string
	Table    string // numeric id or the rt_tables alias, as printed
}

// MatchesTable reports whether the rule selects the given table, accepting both
// the numeric id and our registered rt_tables alias — iproute2 prints the alias
// once /etc/iproute2/rt_tables.d names the id.
func (p PolicyRule) MatchesTable(table int) bool {
	return p.Table == strconv.Itoa(table) || p.Table == naming.RoutingTableAlias
}

// ParsePolicyRules extracts the rules from `ip rule show` output.
func ParsePolicyRules(output string) []PolicyRule {
	var rules []PolicyRule
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		rule := PolicyRule{}
		if strings.HasSuffix(fields[0], ":") {
			rule.Priority = strings.TrimSuffix(fields[0], ":")
		}
		for i, f := range fields {
			if i+1 >= len(fields) {
				break
			}
			switch f {
			case "oif":
				rule.OIF = fields[i+1]
			case "lookup":
				rule.Table = fields[i+1]
			}
		}
		if rule.Table == "" {
			continue
		}
		rules = append(rules, rule)
	}
	return rules
}

// TableRoute is one parsed line of `ip route show table N`.
type TableRoute struct {
	Destination string
	Device      string
}

// ParseTableRoutes extracts destination/device pairs from `ip route show table N`.
func ParseTableRoutes(output string) []TableRoute {
	var routes []TableRoute
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		route := TableRoute{Destination: fields[0]}
		for i, f := range fields {
			if f == "dev" && i+1 < len(fields) {
				route.Device = fields[i+1]
				break
			}
		}
		routes = append(routes, route)
	}
	return routes
}
