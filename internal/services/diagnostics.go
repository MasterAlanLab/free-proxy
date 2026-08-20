package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/naming"
	"github.com/masteralanlab/free-proxy/internal/netx"
)

// DiagnosticsService runs system readiness checks and DNS repair.
type DiagnosticsService struct {
	cfg    *config.Config
	runner netx.CommandRunner

	mu                sync.Mutex
	lastProviderCheck []domain.DiagnosticCheck
}

// NewDiagnosticsService constructs a DiagnosticsService.
func NewDiagnosticsService(cfg *config.Config, runner netx.CommandRunner) *DiagnosticsService {
	if runner == nil {
		runner = netx.SystemCommandRunner{}
	}
	return &DiagnosticsService{cfg: cfg, runner: runner}
}

// AutoRepairEnabled reports whether automatic DNS repair is enabled.
func (d *DiagnosticsService) AutoRepairEnabled() bool { return d.cfg.DNSRepairEnabled }

func linux() bool { return runtime.GOOS == "linux" }

// Diagnose runs all readiness checks.
func (d *DiagnosticsService) Diagnose(ctx context.Context, forStartup bool) domain.SystemDiagnostics {
	openvpn := "openvpn"
	if f := strings.Fields(d.cfg.OpenVPNCommand); len(f) > 0 {
		openvpn = f[0]
	}
	checks := []domain.DiagnosticCheck{
		{Name: "platform", OK: linux(), Detail: runtime.GOOS},
		d.rootCheck(),
		d.dataDirCheck(),
		commandCheck("openvpn", openvpn),
		commandCheck("ip", "ip"),
		commandCheck("sysctl", "sysctl"),
		{Name: "tun_device", OK: fileExists("/dev/net/tun"), Detail: "/dev/net/tun"},
		d.tunAccessCheck(),
		d.tunNameCheck(),
		d.routingTableCheck(ctx),
		d.defaultRouteCheck(ctx),
		d.sysctlCheck(ctx, "ipv4_forwarding", "net.ipv4.ip_forward", "1"),
		d.sysctlCheck(ctx, "rp_filter", "net.ipv4.conf.all.rp_filter", "0", "2"),
		d.dnsCheck(providerHost(d.cfg.VPNGateAPIURL)),
		d.portCheck(forStartup),
	}
	d.mu.Lock()
	checks = append(checks, d.lastProviderCheck...)
	d.mu.Unlock()

	healthy := true
	for _, c := range checks {
		if !c.OK {
			healthy = false
			break
		}
	}
	return domain.SystemDiagnostics{Healthy: healthy, Checks: checks}
}

// StartupPreflight runs checks intended to gate service startup.
func (d *DiagnosticsService) StartupPreflight(ctx context.Context) domain.SystemDiagnostics {
	_ = d.cfg.EnsureDirectories()
	return d.Diagnose(ctx, true)
}

// RepairDNS reconfigures systemd-resolved DNS for the default interface.
func (d *DiagnosticsService) RepairDNS(ctx context.Context) (domain.DnsRepairResult, error) {
	if !d.cfg.DNSRepairEnabled {
		return domain.DnsRepairResult{}, fmt.Errorf("%w: DNS repair is disabled", domain.ErrNetworkOperation)
	}
	if !linux() {
		return domain.DnsRepairResult{}, fmt.Errorf("%w: DNS repair is Linux-only", domain.ErrNetworkOperation)
	}
	if _, err := exec.LookPath("resolvectl"); err != nil {
		return domain.DnsRepairResult{}, fmt.Errorf("%w: resolvectl is required", domain.ErrNetworkOperation)
	}
	route, err := d.runner.Run(ctx, []string{"ip", "route", "show", "default"}, 5*time.Second)
	iface := ParseDefaultInterface(route.Stdout)
	if err != nil || route.ReturnCode != 0 || iface == "" {
		return domain.DnsRepairResult{}, fmt.Errorf("%w: cannot determine default interface", domain.ErrNetworkOperation)
	}
	servers := d.cfg.ParsedDNSRepairServers()
	if len(servers) == 0 {
		return domain.DnsRepairResult{}, fmt.Errorf("%w: no DNS repair servers configured", domain.ErrNetworkOperation)
	}
	commands := [][]string{
		append([]string{"resolvectl", "dns", iface}, servers...),
		{"resolvectl", "domain", iface, "~."},
		{"resolvectl", "flush-caches"},
	}
	for _, cmd := range commands {
		res, err := d.runner.Run(ctx, cmd, 5*time.Second)
		if err != nil || res.ReturnCode != 0 {
			return domain.DnsRepairResult{}, fmt.Errorf("%w: %s failed", domain.ErrNetworkOperation, strings.Join(cmd, " "))
		}
	}
	return domain.DnsRepairResult{Repaired: true, Interface: iface, Servers: servers, Detail: "systemd-resolved DNS configuration updated"}, nil
}

func (d *DiagnosticsService) rootCheck() domain.DiagnosticCheck {
	if !linux() {
		return domain.DiagnosticCheck{Name: "root", OK: true, Detail: "not required outside Linux", Severity: "warning"}
	}
	uid := os.Geteuid()
	return domain.DiagnosticCheck{Name: "root", OK: uid == 0, Detail: fmt.Sprintf("uid=%d", uid)}
}

func (d *DiagnosticsService) dataDirCheck() domain.DiagnosticCheck {
	if err := d.cfg.EnsureDirectories(); err != nil {
		return domain.DiagnosticCheck{Name: "data_directory", OK: false, Detail: err.Error()}
	}
	writable := canWrite(d.cfg.DataDir)
	return domain.DiagnosticCheck{Name: "data_directory", OK: writable, Detail: d.cfg.DataDir}
}

func (d *DiagnosticsService) tunAccessCheck() domain.DiagnosticCheck {
	if !fileExists("/dev/net/tun") {
		return domain.DiagnosticCheck{Name: "tun_access", OK: false, Detail: "/dev/net/tun is missing"}
	}
	return domain.DiagnosticCheck{Name: "tun_access", OK: canWrite("/dev/net/tun"), Detail: "read/write access"}
}

// tunNameCheck reports whether the configured tunnel device name is claimed by
// a program outside this project — the collision behind issue #2. A device that
// is inside our own prefix pool is our tunnel or a leftover we reclaim, not a
// conflict, so it must not be flagged.
func (d *DiagnosticsService) tunNameCheck() domain.DiagnosticCheck {
	if !linux() {
		return domain.DiagnosticCheck{Name: "tun_name", OK: true, Detail: "not applicable outside Linux", Severity: "warning"}
	}
	device, prefix := d.cfg.TunnelInterface, d.cfg.ProbeDevicePrefix
	if naming.HasDevicePrefix(device, prefix) || !netx.DeviceExists(device) {
		return domain.DiagnosticCheck{Name: "tun_name", OK: true, Detail: device}
	}
	return domain.DiagnosticCheck{Name: "tun_name", OK: false, Recoverable: true, Detail: fmt.Sprintf(
		"%s is already owned by another program; set FREE_PROXY_TUNNEL_INTERFACE to an unused name", device)}
}

// routingTableCheck reports whether another program shares our policy routing
// table id. Not fatal (teardown is attribution-based) but worth surfacing.
func (d *DiagnosticsService) routingTableCheck(ctx context.Context) domain.DiagnosticCheck {
	if !linux() {
		return domain.DiagnosticCheck{Name: "routing_table", OK: true, Detail: "not applicable outside Linux", Severity: "warning"}
	}
	router := netx.NewPolicyRouter(d.runner, netx.PolicyRouterConfig{
		Table: d.cfg.PolicyRoutingTable, Interface: d.cfg.TunnelInterface, DevicePrefix: d.cfg.ProbeDevicePrefix,
	})
	if foreign := router.TableConflict(ctx); foreign > 0 {
		return domain.DiagnosticCheck{Name: "routing_table", OK: false, Recoverable: true, Detail: fmt.Sprintf(
			"table %d holds %d route(s) from another program; set FREE_PROXY_POLICY_ROUTING_TABLE to a free id",
			d.cfg.PolicyRoutingTable, foreign)}
	}
	return domain.DiagnosticCheck{Name: "routing_table", OK: true, Detail: fmt.Sprintf("table %d", d.cfg.PolicyRoutingTable)}
}

func (d *DiagnosticsService) defaultRouteCheck(ctx context.Context) domain.DiagnosticCheck {
	res, err := d.runner.Run(ctx, []string{"ip", "route", "show", "default"}, 5*time.Second)
	iface := ParseDefaultInterface(res.Stdout)
	ok := err == nil && res.ReturnCode == 0 && iface != ""
	detail := iface
	if detail == "" {
		detail = strings.TrimSpace(res.Stderr)
		if detail == "" {
			detail = "no default route"
		}
	}
	return domain.DiagnosticCheck{Name: "default_route", OK: ok, Detail: detail}
}

func (d *DiagnosticsService) sysctlCheck(ctx context.Context, name, key string, allowed ...string) domain.DiagnosticCheck {
	res, err := d.runner.Run(ctx, []string{"sysctl", "-n", key}, 5*time.Second)
	value := strings.TrimSpace(res.Stdout)
	ok := err == nil && res.ReturnCode == 0
	if ok {
		match := false
		for _, a := range allowed {
			if value == a {
				match = true
				break
			}
		}
		ok = match
	}
	detail := key + "=" + value
	if value == "" {
		detail = key + "=" + strings.TrimSpace(res.Stderr)
	}
	return domain.DiagnosticCheck{Name: name, OK: ok, Detail: detail, Recoverable: true}
}

// rpFilterCheck reports whether reverse-path filtering would drop the tunnel's
// return traffic.
//
// It reads the *effective* value, which the kernel defines as
// max(conf.all, conf.<iface>) — not conf.all alone. Checking conf.all was
// correct only while we forced it to 2; now that we scope our change to our own
// device (so we stop weakening every other interface on the host), a host
// shipping conf.all=1 would fail this check forever despite working perfectly.
func (d *DiagnosticsService) rpFilterCheck(ctx context.Context) domain.DiagnosticCheck {
	const name = "rp_filter"
	if !linux() {
		return domain.DiagnosticCheck{Name: name, OK: true, Detail: "not applicable outside Linux", Severity: "warning"}
	}
	device := d.cfg.TunnelInterface
	if !netx.DeviceExists(device) {
		// Nothing to judge: we configure the device when a tunnel connects.
		return domain.DiagnosticCheck{Name: name, OK: true, Severity: "warning",
			Detail: device + " is not up; configured when a tunnel connects"}
	}
	all, allOK := d.readSysctlInt(ctx, "net.ipv4.conf.all.rp_filter")
	dev, devOK := d.readSysctlInt(ctx, "net.ipv4.conf."+device+".rp_filter")
	if !allOK || !devOK {
		return domain.DiagnosticCheck{Name: name, OK: false, Recoverable: true, Detail: "could not read rp_filter"}
	}
	detail := fmt.Sprintf("%s effective=%d (all=%d, %s=%d)", device, max(all, dev), all, device, dev)
	if !rpFilterOK(all, dev) {
		return domain.DiagnosticCheck{Name: name, OK: false, Recoverable: true,
			Detail: detail + "; strict mode drops tunnel return traffic"}
	}
	return domain.DiagnosticCheck{Name: name, OK: true, Detail: detail}
}

// rpFilterOK reports whether the effective reverse-path mode lets tunnel return
// traffic through. The kernel applies max(conf.all, conf.<iface>); 0 (disabled)
// and 2 (loose) both pass, and only 1 (strict) drops our traffic, because the
// reverse lookup consults the main table where no route points back out of the
// tunnel.
func rpFilterOK(all, dev int) bool { return max(all, dev) != 1 }

func (d *DiagnosticsService) readSysctlInt(ctx context.Context, key string) (int, bool) {
	res, err := d.runner.Run(ctx, []string{"sysctl", "-n", key}, 5*time.Second)
	if err != nil || res.ReturnCode != 0 {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimSpace(res.Stdout))
	if err != nil {
		return 0, false
	}
	return value, true
}

func (d *DiagnosticsService) portCheck(requireAvailable bool) domain.DiagnosticCheck {
	addr := net.JoinHostPort(d.cfg.ProxyHost, fmt.Sprintf("%d", d.cfg.ProxyPort))
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		_ = ln.Close()
		return domain.DiagnosticCheck{Name: "proxy_port", OK: true, Detail: addr + " available", Recoverable: true}
	}
	if !requireAvailable {
		conn, derr := net.DialTimeout("tcp", addr, time.Second)
		if derr == nil {
			_ = conn.Close()
			return domain.DiagnosticCheck{Name: "proxy_port", OK: true, Detail: addr + " accepting connections"}
		}
	}
	return domain.DiagnosticCheck{Name: "proxy_port", OK: false, Detail: err.Error(), Recoverable: true}
}

func (d *DiagnosticsService) dnsCheck(host string) domain.DiagnosticCheck {
	if host == "" {
		return domain.DiagnosticCheck{Name: "provider_dns", OK: false, Detail: "invalid provider URL"}
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return domain.DiagnosticCheck{Name: "provider_dns", OK: false, Detail: err.Error()}
	}
	sort.Strings(addrs)
	return domain.DiagnosticCheck{Name: "provider_dns", OK: len(addrs) > 0, Detail: strings.Join(addrs, ", ")}
}

// DiagnoseProviderFailure records provider-connectivity checks after a failure.
func (d *DiagnosticsService) DiagnoseProviderFailure(ctx context.Context, cause error) []domain.DiagnosticCheck {
	host := providerHost(d.cfg.VPNGateAPIURL)
	checks := []domain.DiagnosticCheck{
		{Name: "provider_last_error", OK: false, Detail: cause.Error()},
		d.dnsCheck(host),
		tcpCheck("provider_tcp", host, 443),
		tcpCheck("external_network", "1.1.1.1", 443),
		tlsCheck(host),
	}
	d.mu.Lock()
	d.lastProviderCheck = checks
	d.mu.Unlock()
	return checks
}

// ClearProviderFailure clears the recorded provider failure checks.
func (d *DiagnosticsService) ClearProviderFailure() {
	d.mu.Lock()
	d.lastProviderCheck = nil
	d.mu.Unlock()
}

func commandCheck(name, executable string) domain.DiagnosticCheck {
	path, err := exec.LookPath(executable)
	if err != nil {
		return domain.DiagnosticCheck{Name: name, OK: false, Detail: "not found"}
	}
	return domain.DiagnosticCheck{Name: name, OK: true, Detail: path}
}

func tcpCheck(name, host string, port int) domain.DiagnosticCheck {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)), 2*time.Second)
	if err != nil {
		return domain.DiagnosticCheck{Name: name, OK: false, Detail: err.Error()}
	}
	_ = conn.Close()
	return domain.DiagnosticCheck{Name: name, OK: true, Detail: fmt.Sprintf("%s:%d reachable", host, port)}
}

func tlsCheck(host string) domain.DiagnosticCheck {
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, "443"), &tls.Config{ServerName: host})
	if err != nil {
		return domain.DiagnosticCheck{Name: "provider_tls", OK: false, Detail: err.Error()}
	}
	_ = conn.Close()
	return domain.DiagnosticCheck{Name: "provider_tls", OK: true, Detail: "TLS handshake succeeded"}
}

func providerHost(rawURL string) string {
	u := rawURL
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	}
	if i := strings.IndexAny(u, "/:"); i >= 0 {
		u = u[:i]
	}
	return u
}

// ParseDefaultInterface extracts the device from `ip route show default` output.
func ParseDefaultInterface(routeOutput string) string {
	parts := strings.Fields(routeOutput)
	for i, p := range parts {
		if p == "dev" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// IsDNSError reports whether an error looks like a DNS resolution failure.
func IsDNSError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	for _, m := range []string{"dns", "name or service not known", "temporary failure in name resolution",
		"nodename nor servname provided", "cannot resolve", "no such host", "server misbehaving"} {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func canWrite(path string) bool {
	// Best-effort writability probe: stat + attempt to open the dir/file for write is
	// intrusive, so we rely on stat success plus mode bits where available.
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		f, err := os.CreateTemp(path, ".fpwrite-*")
		if err != nil {
			return false
		}
		name := f.Name()
		_ = f.Close()
		_ = os.Remove(name)
		return true
	}
	return info.Mode().Perm()&0o600 != 0
}
