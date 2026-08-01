// Package vpngate discovers OpenVPN nodes from the public VPNGate endpoint.
package vpngate

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
)

// ParseStats records CSV parsing counters for diagnostics.
type ParseStats struct {
	TotalRows        int
	ValidRows        int
	DuplicateRows    int
	MalformedRows    int
	MissingFieldRows int
}

// ParseResult bundles parsed nodes and stats.
type ParseResult struct {
	Nodes []domain.DiscoveredNode
	Stats ParseStats
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

func safeInt(v string) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0
	}
	return n
}

func normalizeTransport(v string) domain.TransportProtocol {
	l := strings.ToLower(v)
	switch {
	case strings.Contains(l, "tcp"):
		return domain.TransportTCP
	case strings.Contains(l, "udp"):
		return domain.TransportUDP
	default:
		return domain.TransportUnknown
	}
}

func parseRemote(configText, fallbackHost string) (string, int, domain.TransportProtocol) {
	remoteHost := fallbackHost
	remotePort := 0
	transport := domain.TransportUnknown
	for _, raw := range strings.Split(configText, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		switch strings.ToLower(parts[0]) {
		case "proto":
			if len(parts) >= 2 {
				transport = normalizeTransport(parts[1])
			}
		case "remote":
			if len(parts) >= 3 {
				remoteHost = parts[1]
				remotePort = safeInt(parts[2])
				if len(parts) >= 4 {
					transport = normalizeTransport(parts[3])
				}
			}
		}
	}
	return remoteHost, remotePort, transport
}

func decodeConfig(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		// try raw/url variants leniently
		data, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(encoded))
		if err != nil {
			return "", fmt.Errorf("invalid OpenVPN configuration")
		}
	}
	return string(data), nil
}

func buildNodeID(countryCode, ipAddress string) string {
	sum := sha256.Sum256([]byte("vpngate:" + ipAddress))
	digest := hex.EncodeToString(sum[:])[:12]
	prefix := nonAlnum.ReplaceAllString(strings.ToUpper(countryCode), "")
	if prefix == "" {
		prefix = "XX"
	}
	return strings.ToLower(prefix) + "-" + digest
}

func providerIdentity(ipAddress string) string {
	return "vpngate:" + strings.TrimSpace(ipAddress)
}

// ParseResponse parses the VPNGate CSV, keeping at most limit nodes.
func ParseResponse(text string, limit int, now time.Time) (ParseResult, error) {
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimRight(l, "\r")
		if l == "" || strings.HasPrefix(l, "*") {
			continue
		}
		lines = append(lines, l)
	}
	if len(lines) == 0 {
		return ParseResult{}, fmt.Errorf("VPNGate response is empty or has no header")
	}
	lines[0] = strings.TrimPrefix(lines[0], "#")
	if !strings.Contains(lines[0], "IP") || !strings.Contains(lines[0], "OpenVPN_ConfigData_Base64") {
		return ParseResult{}, fmt.Errorf("VPNGate response has no usable header")
	}

	reader := csv.NewReader(strings.NewReader(strings.Join(lines, "\n")))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return ParseResult{}, fmt.Errorf("VPNGate header unreadable: %w", err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.TrimSpace(h)] = i
	}
	get := func(row []string, name string) string {
		if idx, ok := col[name]; ok && idx < len(row) {
			return row[idx]
		}
		return ""
	}

	var result ParseResult
	seen := map[string]bool{}
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return ParseResult{}, fmt.Errorf("VPNGate CSV row unreadable: %w", err)
		}
		result.Stats.TotalRows++
		if len(result.Nodes) >= limit {
			continue
		}
		ipAddress := strings.TrimSpace(get(row, "IP"))
		encoded := strings.TrimSpace(get(row, "OpenVPN_ConfigData_Base64"))
		if ipAddress == "" || encoded == "" {
			result.Stats.MissingFieldRows++
			continue
		}
		configText, err := decodeConfig(encoded)
		if err != nil {
			result.Stats.MalformedRows++
			continue
		}
		remoteHost, remotePort, transport := parseRemote(configText, ipAddress)
		if remotePort == 0 {
			result.Stats.MalformedRows++
			continue
		}
		identity := providerIdentity(ipAddress)
		if seen[identity] {
			result.Stats.DuplicateRows++
			continue
		}
		countryCode := strings.ToUpper(strings.TrimSpace(get(row, "CountryShort")))
		result.Nodes = append(result.Nodes, domain.DiscoveredNode{
			ID:               buildNodeID(countryCode, ipAddress),
			Provider:         "vpngate",
			ProviderIdentity: identity,
			ProviderNodeID:   get(row, "HostName"),
			Country:          get(row, "CountryLong"),
			CountryCode:      countryCode,
			HostName:         get(row, "HostName"),
			IPAddress:        ipAddress,
			RemoteHost:       remoteHost,
			RemotePort:       remotePort,
			Transport:        transport,
			SourceScore:      safeInt(get(row, "Score")),
			SourcePingMS:     safeInt(get(row, "Ping")),
			SourceSpeedBPS:   int64(safeInt(get(row, "Speed"))),
			SourceSessions:   safeInt(get(row, "NumVpnSessions")),
			ConfigText:       configText,
			FetchedAt:        now,
		})
		seen[identity] = true
		result.Stats.ValidRows++
	}
	return result, nil
}
