package tunnel

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
)

// ring is a fixed-size ring buffer of recent log lines.
type ring struct {
	mu    sync.Mutex
	buf   []string
	limit int
}

func newRing(limit int) *ring { return &ring{limit: limit} }

func (r *ring) append(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf = append(r.buf, line)
	if len(r.buf) > r.limit {
		r.buf = r.buf[len(r.buf)-r.limit:]
	}
}

func (r *ring) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.buf))
	copy(out, r.buf)
	return out
}

// StartParams configures a process start.
type StartParams struct {
	Bin            string
	Args           []string
	ConfigPath     string
	Device         string
	StartupTimeout time.Duration
	KeepAlive      bool
}

// Runner starts OpenVPN processes and waits for the handshake.
type Runner struct{}

// Start launches OpenVPN, reads its output until ready/failure/timeout, and
// (when KeepAlive) returns a supervised Managed process.
func (Runner) Start(ctx context.Context, p StartParams) (domain.TunnelStartResult, *Managed) {
	started := time.Now()
	pr, pw, err := os.Pipe()
	if err != nil {
		return failResult(domain.FailStartFailed, "OpenVPN output pipe could not be created", started, nil), nil
	}
	cmd := exec.Command(p.Bin, p.Args...)
	cmd.Stdout = pw
	cmd.Stderr = pw
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		code := domain.FailStartFailed
		msg := fmt.Sprintf("OpenVPN could not be started: %v", err)
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			code = domain.FailCommandNotFound
			msg = "OpenVPN executable was not found"
		}
		return failResult(code, msg, started, nil), nil
	}
	_ = pw.Close() // parent drops its write end so pr sees EOF on child exit

	tail := newRing(50)
	readyCh := make(chan struct{}, 1)
	termFailCh := make(chan struct{}, 1)
	scanDone := make(chan struct{})

	go func() {
		defer close(scanDone)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
		for sc.Scan() {
			line := sc.Text()
			tail.append(line)
			slog.Info(line, "module", "openvpn")
			if IsReady(line) {
				select {
				case readyCh <- struct{}{}:
				default:
				}
				continue
			}
			if IsTerminalFailure(line) {
				select {
				case termFailCh <- struct{}{}:
				default:
				}
			}
		}
	}()

	timer := time.NewTimer(p.StartupTimeout)
	defer timer.Stop()

	select {
	case <-readyCh:
		startupMS := int(time.Since(started).Milliseconds())
		if !p.KeepAlive {
			terminateAndWait(cmd)
			_ = pr.Close()
			return domain.TunnelStartResult{
				Success: true, Status: domain.TunnelConnected, Message: "Tunnel handshake completed",
				StartupTimeMS: startupMS, LogTail: tail.snapshot(), HandshakeStage: "connected",
			}, nil
		}
		m := newManaged(cmd, pr, p.ConfigPath, p.Device, tail)
		return domain.TunnelStartResult{
			Success: true, Status: domain.TunnelConnected, Message: "Tunnel connected",
			StartupTimeMS: startupMS, LogTail: tail.snapshot(), HandshakeStage: "connected",
		}, m
	case <-termFailCh:
	case <-timer.C:
		tail.append(fmt.Sprintf("OpenVPN timeout after %.1fs", p.StartupTimeout.Seconds()))
	case <-scanDone:
	case <-ctx.Done():
	}

	terminateAndWait(cmd)
	_ = pr.Close()
	lines := tail.snapshot()
	return domain.TunnelStartResult{
		Success:        false,
		Status:         domain.TunnelFailed,
		Message:        FailureMessageFor(lines, p.Device),
		FailureCode:    ptr(FailureCode(lines)),
		StartupTimeMS:  int(time.Since(started).Milliseconds()),
		LogTail:        lines,
		HandshakeStage: HandshakeStage(lines),
	}, nil
}

func failResult(code domain.TunnelFailureCode, msg string, started time.Time, tail []string) domain.TunnelStartResult {
	return domain.TunnelStartResult{
		Success:        false,
		Status:         domain.TunnelFailed,
		Message:        msg,
		FailureCode:    ptr(code),
		StartupTimeMS:  int(time.Since(started).Milliseconds()),
		LogTail:        tail,
		HandshakeStage: "starting",
	}
}

func ptr[T any](v T) *T { return &v }

// Managed is a supervised, running OpenVPN process.
type Managed struct {
	cmd        *exec.Cmd
	pr         *os.File
	configPath string
	device     string
	tail       *ring

	mu          sync.Mutex
	intentional bool
	onExit      func(code int)

	running atomic.Bool
	done    chan struct{}
}

func newManaged(cmd *exec.Cmd, pr *os.File, configPath, device string, tail *ring) *Managed {
	m := &Managed{cmd: cmd, pr: pr, configPath: configPath, device: device, tail: tail, done: make(chan struct{})}
	m.running.Store(true)
	go m.watch()
	return m
}

func (m *Managed) watch() {
	err := m.cmd.Wait()
	m.running.Store(false)
	close(m.done)
	m.mu.Lock()
	intentional := m.intentional
	handler := m.onExit
	m.mu.Unlock()
	if !intentional && handler != nil {
		handler(exitCode(err))
	}
}

// Running reports whether the process is still alive.
func (m *Managed) Running() bool { return m.running.Load() }

// Device returns the tunnel device name.
func (m *Managed) Device() string { return m.device }

// SetExitHandler installs a callback invoked on unexpected exit.
func (m *Managed) SetExitHandler(h func(code int)) {
	m.mu.Lock()
	m.onExit = h
	m.mu.Unlock()
}

// Stop terminates the process group and removes the config file. Safe to call
// on a process that already exited.
func (m *Managed) Stop() {
	m.mu.Lock()
	m.intentional = true
	m.mu.Unlock()
	// Once watch() has reaped the child, the kernel is free to hand its pid to
	// somebody else — and we kill by *process group*, so a stale pgid would
	// signal a stranger's whole group. An exited process needs no signal
	// anyway. This path is reached routinely: Disconnect and Connect both call
	// Stop on a tunnel that may have died on its own moments earlier.
	if m.cmd.Process != nil && !m.exited() {
		pgid := m.cmd.Process.Pid
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
		select {
		case <-m.done:
		case <-time.After(8 * time.Second):
			// Re-check: the process may have exited during the grace period,
			// making the pgid reusable before we escalate.
			if !m.exited() {
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			}
			<-m.done
		}
	}
	_ = m.pr.Close()
	if m.configPath != "" {
		_ = os.Remove(m.configPath)
	}
}

// exited reports whether watch() has already reaped the child, after which its
// pid must not be signalled.
func (m *Managed) exited() bool {
	select {
	case <-m.done:
		return true
	default:
		return false
	}
}

func terminateAndWait(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pgid := cmd.Process.Pid
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		<-done
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}
