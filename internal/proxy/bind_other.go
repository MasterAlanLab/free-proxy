//go:build !linux

package proxy

import "syscall"

// bindControl is a no-op on non-Linux platforms (SO_BINDTODEVICE is
// Linux-only). This lets the proxy build and its end-to-end tests run on
// macOS/dev machines; real interface binding only happens on Linux.
func bindControl(iface string) func(network, address string, c syscall.RawConn) error {
	return nil
}
