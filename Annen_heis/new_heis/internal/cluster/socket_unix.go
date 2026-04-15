//go:build darwin || linux

package cluster

import (
	"context"
	"fmt"
	"net"
	"syscall"
)

func listenBroadcastUDP(port int) (net.PacketConn, error) {
	config := net.ListenConfig{
		Control: func(network, address string, rawConn syscall.RawConn) error {
			var controlErr error
			err := rawConn.Control(func(fd uintptr) {
				if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
					controlErr = err
					return
				}
				_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
				_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
			})
			if err != nil {
				return err
			}
			return controlErr
		},
	}
	return config.ListenPacket(context.Background(), "udp4", fmt.Sprintf(":%d", port))
}
