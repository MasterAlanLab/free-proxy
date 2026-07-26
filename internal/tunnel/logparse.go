// Package tunnel manages the OpenVPN process lifecycle: building the command,
// classifying log output, and running/supervising the process.
package tunnel

import (
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
	case anyOf("options error", "fatal error", "unrecognized option", "option error"):
		return domain.FailConfigError
	case anyOf("network is unreachable"):
		return domain.FailUnreachable
	case anyOf("timed out", "timeout"):
		return domain.FailTimeout
	default:
		return domain.FailUnknown
	}
}

// FailureMessage returns a human-readable message for the classified failure.
func FailureMessage(lines []string) string {
	messages := map[domain.TunnelFailureCode]string{
		domain.FailTunUnavailable:   "TUN device is unavailable or not permitted",
		domain.FailAuthFailed:       "The public node rejected authentication",
		domain.FailDNSFailed:        "The public node address could not be resolved",
		domain.FailTLSFailed:        "OpenVPN TLS negotiation failed",
		domain.FailPermissionDenied: "OpenVPN requires elevated network permissions",
		domain.FailConfigError:      "The OpenVPN configuration contains an invalid option",
		domain.FailConnectionRefused: "The public node refused the connection",
		domain.FailUnreachable:      "The public node is unreachable",
		domain.FailTimeout:          "OpenVPN connection timed out",
		domain.FailUnknown:          "OpenVPN failed before the tunnel became ready",
	}
	if msg, ok := messages[FailureCode(lines)]; ok {
		return msg
	}
	return messages[domain.FailUnknown]
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
