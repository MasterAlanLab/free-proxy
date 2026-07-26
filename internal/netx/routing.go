package netx

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// PolicyRouter installs the policy route/rule that forces marked traffic through
// the tunnel and relaxes rp_filter, restoring prior values on cleanup. Linux only.
type PolicyRouter struct {
	runner        CommandRunner
	table         int
	iface         string
	setupRetries  int
	retryInterval time.Duration
	strictRPF     bool

	rpFilterOriginal map[string]string
}

// PolicyRouterConfig carries the settings a PolicyRouter needs.
type PolicyRouterConfig struct {
	Table         int
	Interface     string
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
	return &PolicyRouter{
		runner:           runner,
		table:            cfg.Table,
		iface:            cfg.Interface,
		setupRetries:     cfg.SetupRetries,
		retryInterval:    cfg.RetryInterval,
		strictRPF:        cfg.StrictRPF,
		rpFilterOriginal: map[string]string{},
	}
}

// Supported reports whether policy routing is available on this OS.
func (r *PolicyRouter) Supported() bool { return runtime.GOOS == "linux" }

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
	for _, target := range []string{"all", "default", device} {
		key := "net.ipv4.conf." + target + ".rp_filter"
		if read, err := r.runner.Run(ctx, []string{"sysctl", "-n", key}, 5*time.Second); err == nil && read.ReturnCode == 0 {
			r.rpFilterOriginal[target] = strings.TrimSpace(read.Stdout)
		}
		set, err := r.runner.Run(ctx, []string{"sysctl", "-w", key + "=2"}, 5*time.Second)
		if err != nil || set.ReturnCode != 0 {
			msg := fmt.Sprintf("unable to configure rp_filter for %s", target)
			if r.strictRPF {
				return fmt.Errorf("%s", msg)
			}
			slog.Warn(msg, "module", "netx")
		}
	}
	return nil
}

// Cleanup removes the route/rule and restores rp_filter values.
func (r *PolicyRouter) Cleanup(ctx context.Context) error {
	if !r.Supported() {
		return nil
	}
	table := strconv.Itoa(r.table)
	_, _ = r.runner.Run(ctx, []string{"ip", "rule", "del", "table", table}, 5*time.Second)
	_, _ = r.runner.Run(ctx, []string{"ip", "route", "flush", "table", table}, 5*time.Second)
	for target, value := range r.rpFilterOriginal {
		_, _ = r.runner.Run(ctx, []string{"sysctl", "-w", "net.ipv4.conf." + target + ".rp_filter=" + value}, 5*time.Second)
	}
	r.rpFilterOriginal = map[string]string{}
	return nil
}
