//go:build linux

package proxy

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// bindControl returns a net.Dialer.Control callback that binds the outbound
// socket to the given interface via SO_BINDTODEVICE, forcing traffic through the
// tunnel. An empty iface disables binding. Requires CAP_NET_RAW (root).
func bindControl(iface string) func(network, address string, c syscall.RawConn) error {
	if iface == "" {
		return nil
	}
	return func(_, _ string, c syscall.RawConn) error {
		var setErr error
		if err := c.Control(func(fd uintptr) {
			setErr = unix.SetsockoptString(int(fd), unix.SOL_SOCKET, unix.SO_BINDTODEVICE, iface)
		}); err != nil {
			return err
		}
		return setErr
	}
}
