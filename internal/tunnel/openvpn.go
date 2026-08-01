package tunnel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
)

var versionRe = regexp.MustCompile(`OpenVPN\s+(\d+)\.(\d+)`)

// Manager owns the active OpenVPN tunnel and runs probe dials.
type Manager struct {
	cfg    *config.Config
	runner Runner

	authFile string

	mu           sync.Mutex
	active       *Managed
	activeNodeID string
	version      *Version
	exitHandler  func(code int)
}

// NewManager constructs a Manager.
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg:      cfg,
		authFile: filepath.Join(cfg.DataDir, "openvpn-auth.txt"),
	}
}

// ActiveNodeID returns the id of the currently connected node, if any.
func (m *Manager) ActiveNodeID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeNodeID
}

// ActiveRunning reports whether a tunnel is currently up.
func (m *Manager) ActiveRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active != nil && m.active.Running()
}

// Probe dials a node on a throwaway device to measure connectivity.
func (m *Manager) Probe(ctx context.Context, configText, device string) domain.TunnelStartResult {
	if err := m.ensureAuthFile(); err != nil {
		return failResult(domain.FailStartFailed, "unable to write auth file: "+err.Error(), time.Now(), nil)
	}
	configPath, err := m.writeConfig(configText, "probe")
	if err != nil {
		return failResult(domain.FailStartFailed, "unable to write config: "+err.Error(), time.Now(), nil)
	}
	defer os.Remove(configPath)
	version := m.detectVersion(ctx)
	args := BuildArgs(BuildParams{
		Executable: ParseExecutable(m.cfg.OpenVPNCommand), ConfigFile: configPath, AuthFile: m.authFile,
		Device: device, RouteNoPull: true, Version: version,
	})
	res, _ := m.runner.Start(ctx, StartParams{
		Bin: args[0], Args: args[1:], ConfigPath: configPath, Device: device,
		StartupTimeout: m.cfg.OpenVPNTestTimeout(), KeepAlive: false,
	})
	return res
}

// Connect brings up the active tunnel for a node, replacing any current one.
func (m *Manager) Connect(ctx context.Context, nodeID, configText string) domain.TunnelStartResult {
	// detectVersion acquires m.mu internally; resolve it BEFORE taking the lock
	// here — sync.Mutex is not reentrant, so calling it while holding m.mu would
	// self-deadlock (hanging the connect with no openvpn started).
	version := m.detectVersion(ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectLocked()
	if err := m.ensureAuthFile(); err != nil {
		return failResult(domain.FailStartFailed, "unable to write auth file: "+err.Error(), time.Now(), nil)
	}
	configPath, err := m.writeConfig(configText, "active")
	if err != nil {
		return failResult(domain.FailStartFailed, "unable to write config: "+err.Error(), time.Now(), nil)
	}
	args := BuildArgs(BuildParams{
		Executable: ParseExecutable(m.cfg.OpenVPNCommand), ConfigFile: configPath, AuthFile: m.authFile,
		Device: m.cfg.TunnelInterface, RouteNoPull: true, Version: version,
	})
	res, managed := m.runner.Start(ctx, StartParams{
		Bin: args[0], Args: args[1:], ConfigPath: configPath, Device: m.cfg.TunnelInterface,
		StartupTimeout: m.cfg.OpenVPNConnectTimeout(), KeepAlive: true,
	})
	if res.Success && managed != nil {
		managed.SetExitHandler(m.exitHandler)
		m.active = managed
		m.activeNodeID = nodeID
	} else {
		_ = os.Remove(configPath)
	}
	return res
}

// Disconnect tears down the active tunnel.
func (m *Manager) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectLocked()
}

func (m *Manager) disconnectLocked() {
	if m.active != nil {
		m.active.Stop()
	}
	m.active = nil
	m.activeNodeID = ""
}

// SetExitHandler installs the unexpected-exit callback for future and current tunnels.
func (m *Manager) SetExitHandler(h func(code int)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exitHandler = h
	if m.active != nil {
		m.active.SetExitHandler(h)
	}
}

// ClearExitedProcess drops a reference to a process that already exited.
func (m *Manager) ClearExitedProcess() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil && !m.active.Running() {
		m.active = nil
		m.activeNodeID = ""
	}
}

// CleanupStaleProcesses terminates leftover openvpn processes started by this
// project (identified by config/auth paths in their cmdline). Linux only.
func (m *Manager) CleanupStaleProcesses() []int {
	markers := []string{m.cfg.DataDir, m.authFile}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	self := os.Getpid()
	var terminated []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == self {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil {
			continue
		}
		cmdline := strings.ReplaceAll(string(raw), "\x00", " ")
		if !strings.Contains(strings.ToLower(cmdline), "openvpn") {
			continue
		}
		matched := false
		for _, mk := range markers {
			if strings.Contains(cmdline, mk) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		if p, err := os.FindProcess(pid); err == nil {
			if p.Signal(os.Interrupt) == nil {
				terminated = append(terminated, pid)
			}
		}
	}
	if len(terminated) > 0 {
		time.Sleep(500 * time.Millisecond)
	}
	return terminated
}

func (m *Manager) ensureAuthFile() error {
	if err := m.cfg.EnsureDirectories(); err != nil {
		return err
	}
	data := m.cfg.OpenVPNUsername + "\n" + m.cfg.OpenVPNPassword + "\n"
	return os.WriteFile(m.authFile, []byte(data), 0o600)
}

func (m *Manager) writeConfig(configText, prefix string) (string, error) {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	name := prefix + "-" + hex.EncodeToString(b) + ".ovpn"
	path := filepath.Join(m.cfg.ConfigsDir(), name)
	if err := os.WriteFile(path, []byte(configText), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (m *Manager) detectVersion(ctx context.Context) Version {
	m.mu.Lock()
	if m.version != nil {
		v := *m.version
		m.mu.Unlock()
		return v
	}
	m.mu.Unlock()

	exe := ParseExecutable(m.cfg.OpenVPNCommand)
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, exe[0], append(exe[1:], "--version")...).CombinedOutput()
	v := Version{2, 4}
	if err == nil || len(out) > 0 {
		if mm := versionRe.FindSubmatch(out); mm != nil {
			maj, _ := strconv.Atoi(string(mm[1]))
			min, _ := strconv.Atoi(string(mm[2]))
			v = Version{maj, min}
		}
	}
	m.mu.Lock()
	m.version = &v
	m.mu.Unlock()
	return v
}
