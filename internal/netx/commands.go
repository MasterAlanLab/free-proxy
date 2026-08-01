// Package netx wraps host network operations: running ip/sysctl commands,
// policy routing, latency probes, and the test-time TUN
// device pool. It depends only on config/domain and the standard library.
package netx

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

// CommandResult is the outcome of running an external command.
type CommandResult struct {
	ReturnCode int
	Stdout     string
	Stderr     string
}

// CommandRunner runs external commands; the production impl shells out, tests
// inject a fake.
type CommandRunner interface {
	Run(ctx context.Context, args []string, timeout time.Duration) (CommandResult, error)
}

// SystemCommandRunner executes commands via os/exec.
type SystemCommandRunner struct{}

// Run executes args[0] with args[1:], bounded by timeout.
func (SystemCommandRunner) Run(ctx context.Context, args []string, timeout time.Duration) (CommandResult, error) {
	if len(args) == 0 {
		return CommandResult{}, errors.New("empty command")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, args[0], args[1:]...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	res := CommandResult{Stdout: out.String(), Stderr: errBuf.String()}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ReturnCode = exitErr.ExitCode()
			return res, nil
		}
		if cctx.Err() != nil {
			return res, cctx.Err()
		}
		res.ReturnCode = 127
		if res.Stderr == "" {
			res.Stderr = err.Error()
		}
		return res, nil
	}
	return res, nil
}
