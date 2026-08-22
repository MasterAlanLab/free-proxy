package netx

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/masteralanlab/free-proxy/internal/naming"
)

// AdvertiseAddress is the host chosen for a URL we print for a human to copy.
type AdvertiseAddress struct {
	// Host is the address to put in the URL, or "" when none could be found.
	Host string
	// Public reports whether Host is reachable from outside the machine's own
	// network. A false value is still worth printing (a LAN install is a real
	// use case) but the caller should say so rather than imply internet reach.
	Public bool
}

// publicIPEndpoints are plain-text "what is my IP" services, queried only when
// the box has no public address of its own (NAT'd clouds — Alibaba, Tencent,
// AWS, GCP — give the instance a private address and map an elastic IP to it,
// so the public address simply is not visible from inside). ip-api.com is
// first because the project already depends on it for node classification and
// it answers from networks where the others are slow or blocked.
var publicIPEndpoints = []string{
	"http://ip-api.com/line/?fields=query",
	"https://api.ipify.org",
	"https://ip.sb",
}

// ResolveAdvertiseHost picks the address to show a user for reaching a listener
// bound to bindHost.
//
// A concrete bind address is authoritative and returned as-is. For a wildcard
// bind ("", 0.0.0.0, ::) the listener answers on every interface, so there is
// no single right answer and we pick the most useful one: a public address
// configured on the machine, else whatever a public IP echo service sees us
// as, else a private address so at least a LAN URL works.
//
// It never fails: an empty Host means "unknown, keep your placeholder".
func ResolveAdvertiseHost(ctx context.Context, bindHost string) AdvertiseAddress {
	host := strings.Trim(strings.TrimSpace(bindHost), "[]")
	if host != "" {
		ip := net.ParseIP(host)
		if ip == nil {
			// A hostname was configured deliberately; it is what the operator
			// wants in the URL and we cannot classify it cheaply.
			return AdvertiseAddress{Host: host, Public: true}
		}
		if !ip.IsUnspecified() {
			return AdvertiseAddress{Host: ip.String(), Public: isPublicIP(ip)}
		}
	}

	public, private := localAddresses()
	if public != "" {
		return AdvertiseAddress{Host: public, Public: true}
	}
	if ip := lookupPublicIP(ctx, http.DefaultClient, publicIPEndpoints); ip != "" {
		return AdvertiseAddress{Host: ip, Public: true}
	}
	return AdvertiseAddress{Host: private}
}

// localAddresses returns the best public and best private address configured on
// this host's interfaces, preferring IPv4 in both cases because that is what
// people paste into a browser.
func localAddresses() (public, private string) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", ""
	}
	var publicV6, privateV6 string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isTunnelInterface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP == nil || !ipnet.IP.IsGlobalUnicast() {
				continue
			}
			ip := ipnet.IP
			v4 := ip.To4() != nil
			switch {
			case isPublicIP(ip) && v4 && public == "":
				public = ip.String()
			case isPublicIP(ip) && !v4 && publicV6 == "":
				publicV6 = ip.String()
			case !isPublicIP(ip) && v4 && private == "":
				private = ip.String()
			case !isPublicIP(ip) && !v4 && privateV6 == "":
				privateV6 = ip.String()
			}
		}
	}
	if public == "" {
		public = publicV6
	}
	if private == "" {
		private = privateV6
	}
	return public, private
}

// isTunnelInterface reports whether a device carries somebody's tunnel rather
// than the host's own connectivity. Our own tunnel is the reason this matters:
// on an upgrade the proxy may already be connected, and the VPN exit address
// sitting on that device is public but useless for reaching this machine.
func isTunnelInterface(name string) bool {
	for _, prefix := range []string{naming.DevicePrefix, naming.LegacyDevicePrefix, "tap", "wg", "ppp"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// isPublicIP reports whether an address is routable on the public internet.
func isPublicIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		// 100.64.0.0/10 is carrier-grade NAT: globally unique, never reachable.
		return !(v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127)
	}
	return true
}

// lookupPublicIP asks the endpoints in order and returns the first public
// address any of them reports. The whole sweep is capped so an install never
// hangs on a box with no egress.
func lookupPublicIP(ctx context.Context, client *http.Client, endpoints []string) string {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	for _, endpoint := range endpoints {
		if ip := fetchIP(ctx, client, endpoint); ip != "" {
			return ip
		}
		if ctx.Err() != nil {
			return ""
		}
	}
	return ""
}

func fetchIP(ctx context.Context, client *http.Client, endpoint string) string {
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	// Bounded read: these endpoints answer with a bare address, so anything
	// larger is a captive portal or an error page, not an answer.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return ""
	}
	ip := net.ParseIP(strings.TrimSpace(string(body)))
	if !isPublicIP(ip) {
		return ""
	}
	return ip.String()
}
