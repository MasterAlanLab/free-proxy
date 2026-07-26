package netx

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// UpstreamProxy describes an optional upstream HTTP/SOCKS proxy used when
// fetching provider data and when dialing OpenVPN over TCP.
type UpstreamProxy struct {
	Kind     string // "http" or "socks"
	Host     string
	Port     int
	Username string
	Password string
	HasAuth  bool
}

// URL renders the proxy as a URL string (with credentials when present).
func (u UpstreamProxy) URL() string {
	auth := ""
	if u.HasAuth {
		auth = url.QueryEscape(u.Username) + ":" + url.QueryEscape(u.Password) + "@"
	}
	scheme := "http"
	if u.Kind == "socks" {
		scheme = "socks5"
	}
	host := u.Host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("%s://%s%s:%d", scheme, auth, host, u.Port)
}

// GetUpstreamProxy resolves the configured upstream proxy from the config URL
// or standard proxy environment variables (checked by caller-provided list).
func GetUpstreamProxy(configuredURL string, envValues ...string) *UpstreamProxy {
	candidates := append([]string{configuredURL}, envValues...)
	for _, v := range candidates {
		if p := ParseUpstreamProxy(v, ""); p != nil {
			return p
		}
	}
	return nil
}

// ParseUpstreamProxy parses a proxy URL. forcedKind ("http"/"socks") overrides
// scheme detection when set.
func ParseUpstreamProxy(value, forcedKind string) *UpstreamProxy {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return nil
	}
	if !strings.Contains(normalized, "://") {
		k := forcedKind
		if k == "" {
			k = "http"
		}
		normalized = k + "://" + normalized
	}
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Hostname() == "" {
		return nil
	}
	kind := forcedKind
	if kind == "" {
		if strings.HasPrefix(strings.ToLower(parsed.Scheme), "socks") {
			kind = "socks"
		} else {
			kind = "http"
		}
	}
	defaultPort := 8080
	if kind == "socks" {
		defaultPort = 1080
	}
	port := defaultPort
	if p := parsed.Port(); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}
	out := &UpstreamProxy{Kind: kind, Host: parsed.Hostname(), Port: port}
	if parsed.User != nil {
		out.Username = parsed.User.Username()
		if pw, ok := parsed.User.Password(); ok {
			out.Password = pw
		}
		out.HasAuth = true
	}
	return out
}
