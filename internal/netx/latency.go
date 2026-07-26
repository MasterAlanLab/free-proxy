package netx

import (
	"context"
	"net"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var pingTimeRe = regexp.MustCompile(`time=([\d.]+)\s*ms`)

// MeasureNodeLatency estimates latency to host:port, preferring ICMP ping on the
// physical interface, then plain ping, then a TCP connect, then the fallback.
func MeasureNodeLatency(ctx context.Context, runner CommandRunner, host string, port, fallbackPingMS int) int {
	if runtime.GOOS == "linux" {
		if iface := physicalInterface(ctx, runner); iface != "" {
			if ms := runPing(ctx, runner, host, iface); ms > 0 {
				return ms
			}
		}
	}
	if ms := runPing(ctx, runner, host, ""); ms > 0 {
		return ms
	}
	if ms := measureTCPLatency(host, port, 5*time.Second); ms > 0 {
		return ms
	}
	if fallbackPingMS < 0 {
		return 0
	}
	return fallbackPingMS
}

func physicalInterface(ctx context.Context, runner CommandRunner) string {
	if runtime.GOOS != "linux" {
		return ""
	}
	res, err := runner.Run(ctx, []string{"ip", "route"}, 2*time.Second)
	if err != nil || res.ReturnCode != 0 {
		return ""
	}
	type route struct {
		metric int
		dev    string
	}
	var routes []route
	for _, line := range strings.Split(res.Stdout, "\n") {
		if !strings.HasPrefix(line, "default") {
			continue
		}
		parts := strings.Fields(line)
		dev, metric := "", 0
		for i, p := range parts {
			if p == "dev" && i+1 < len(parts) {
				dev = parts[i+1]
			}
			if p == "metric" && i+1 < len(parts) {
				metric, _ = strconv.Atoi(parts[i+1])
			}
		}
		if dev == "" {
			continue
		}
		if strings.HasPrefix(dev, "tun") || strings.HasPrefix(dev, "tap") ||
			strings.HasPrefix(dev, "wg") || strings.HasPrefix(dev, "ppp") {
			continue
		}
		routes = append(routes, route{metric, dev})
	}
	best := ""
	bestMetric := int(^uint(0) >> 1)
	for _, r := range routes {
		if r.metric < bestMetric {
			bestMetric = r.metric
			best = r.dev
		}
	}
	return best
}

func runPing(ctx context.Context, runner CommandRunner, host, iface string) int {
	args := []string{"ping", "-c", "1", "-W", "2"}
	if iface != "" {
		args = append(args, "-I", iface)
	}
	args = append(args, host)
	res, err := runner.Run(ctx, args, 3*time.Second)
	if err != nil || res.ReturnCode != 0 {
		return 0
	}
	m := pingTimeRe.FindStringSubmatch(res.Stdout)
	if len(m) < 2 {
		return 0
	}
	f, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	if int(f) < 1 {
		return 1
	}
	return int(f)
}

func measureTCPLatency(host string, port int, timeout time.Duration) int {
	if host == "" || port <= 0 {
		return 0
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return 0
	}
	_ = conn.Close()
	ms := int(time.Since(start).Milliseconds())
	if ms < 1 {
		return 1
	}
	return ms
}
