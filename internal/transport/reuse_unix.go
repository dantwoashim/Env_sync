//go:build !windows

package transport

import "syscall"

// reuseControl sets SO_REUSEADDR and SO_REUSEPORT on the socket.
func reuseControl(network, address string, c syscall.RawConn) error {
	var errSet error
	c.Control(func(fd uintptr) {
		errSet = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	})
	return errSet
}
