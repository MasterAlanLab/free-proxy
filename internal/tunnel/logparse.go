// Package tunnel manages the OpenVPN process lifecycle: building the command,
// classifying log output, and running/supervising the process.
package tunnel

import (
	"fmt"
	"strings"

	"github.com/masteralanlab/free-proxy/internal/domain"
)

const readyMarker = "initialization sequence completed"

// IsReady reports whether a log line signals a completed handshake.
func IsReady(line string) bool {
	return strings.Contains(strings.ToLower(line), readyMarker)
}

// FailureCode classifies OpenVPN log output into a TunnelFailureCode.
func FailureCode(lines []string) domain.TunnelFailureCode {
	text := strings.ToLower(strings.Join(lines, "\n"))
	anyOf := func(patterns ...string) bool {
		for _, p := range patterns {
			if strings.Contains(text, p) {
				return true
			}
		}
		return false
	}
	switch {
	case anyOf("cannot allocate tun", "cannot open tun/tap", "/dev/net/tun", "cannot ioctl"):
		return domain.FailTunUnavailable
	case anyOf("auth_failed", "authentication failed"):
		return domain.FailAuthFailed
	case anyOf("cannot resolve host address", "resolve: host name"):
		return domain.FailDNSFailed
	case anyOf("tls key negotiation failed", "tls handshake failed"):
		return domain.FailTLSFailed
	case anyOf("permission denied", "need root", "root privileges", "operation not permitted"):
		return domain.FailPermissionDenied
	case anyOf("connection refused"):
		return domain.FailConnectionRefused
	case hasFatalConfigError(lines):
		return domain.FailConfigError
	case anyOf("network is unreachable"):
		return domain.FailUnreachable
	case anyOf("timed out", "timeout"):
		return domain.FailTimeout
	default:
		return domain.FailUnknown
	}
}

// hasFatalConfigError reports whether the log contains a genuinely fatal
// option/config error. It deliberately ignores the benign
//
//	Options error: option '<x>' cannot be used in this context ([PUSH-OPTIONS])
//
// warnings that every VPNGate node produces: with --route-nopull the client
// intentionally rejects pushed dhcp-option/redirect-gateway directives, logs
// this line, and proceeds to a successful handshake. Classifying those as fatal
// (and therefore terminal) aborts every connection one step before completion.
func hasFatalConfigError(lines []string) bool {
	for _, ln := range lines {
		l := strings.ToLower(ln)
		if strings.Contains(l, "cannot be used in this context") {
			continue // pushed option rejected by the client; non-fatal
		}
		if strings.Contains(l, "options error") ||
			strings.Contains(l, "option error") ||
			strings.Contains(l, "unrecognized option") ||
			strings.Contains(l, "fatal error") {
			return true
		}
	}
	return false
}

var failureMessages = map[domain.TunnelFailureCode]string{
	domain.FailTunUnavailable:    "TUN device is unavailable or not permitted",
	domain.FailAuthFailed:        "The public node rejected authentication",
	domain.FailDNSFailed:         "The public node address could not be resolved",
	domain.FailTLSFailed:         "OpenVPN TLS negotiation failed",
	domain.FailPermissionDenied:  "OpenVPN requires elevated network permissions",
	domain.FailConfigError:       "The OpenVPN configuration contains an invalid option",
	domain.FailConnectionRefused: "The public node refused the connection",
	domain.FailUnreachable:       "The public node is unreachable",
	domain.FailTimeout:           "OpenVPN connection timed out",
	domain.FailUnknown:           "OpenVPN failed before the tunnel became ready",
}

// FailureMessage returns a human-readable message for the classified failure.
func FailureMessage(lines []string) string { return FailureMessageFor(lines, "") }

// FailureMessageFor is FailureMessage with the device name in hand.
//
// A bare "TUN device is unavailable or not permitted" (see issue #2) is the
// hardest failure to act on, because it does not say *which* device or that a
// name collision with another program is the likely cause. When we know the
// device, say so and point at the setting that moves us out of the way.
func FailureMessageFor(lines []string, device string) string {
	code := FailureCode(lines)
	if code == domain.FailTunUnavailable && device != "" {
		return fmt.Sprintf("TUN device %s could not be created: the name may already be taken by another program "+
			"(3x-ui, WARP, another VPN client), or /dev/net/tun is not permitted. "+
			"Set FREE_PROXY_TUNNEL_INTERFACE (active tunnel) or FREE_PROXY_PROBE_DEVICE_PREFIX (probes) to an unused name.", device)
	}
	if msg, ok := failureMessages[code]; ok {
		return msg
	}
	return failureMessages[domain.FailUnknown]
}

// IsTerminalFailure reports whether a single line indicates an unrecoverable
// failure that should abort the handshake wait early.
func IsTerminalFailure(line string) bool {
	switch FailureCode([]string{line}) {
	case domain.FailAuthFailed, domain.FailConfigError,
		domain.FailPermissionDenied, domain.FailTunUnavailable:
		return true
	default:
		return false
	}
}

// HandshakeStage summarizes how far the handshake progressed.
func HandshakeStage(lines []string) string {
	text := strings.ToLower(strings.Join(lines, "\n"))
	switch {
	case strings.Contains(text, readyMarker):
		return "connected"
	case strings.Contains(text, "auth"):
		return "authentication"
	case strings.Contains(text, "tls"):
		return "tls"
	case strings.Contains(text, "tun"), strings.Contains(text, "tap"):
		return "interface"
	case strings.Contains(text, "resolve"):
		return "resolving"
	default:
		return "starting"
	}
}
