package elevator

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/JesperVigtel/Heis_test/elevio"
)

// startMockServer starts a TCP server that implements the 4-byte elevator
// simulator protocol well enough for unit tests. It accepts one connection,
// responds to read commands with safe defaults, and silently discards write
// commands.
func startMockServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("startMockServer: listen: %v", err)
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return // listener was closed
		}
		defer conn.Close()
		buf := make([]byte, 4)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
			switch buf[0] {
			case 6: // GetButton → always not pressed
				_, _ = conn.Write([]byte{6, 0, 0, 0})
			case 7: // GetFloor → always at floor 0
				_, _ = conn.Write([]byte{7, 1, 0, 0})
			case 8: // GetStop → not pressed
				_, _ = conn.Write([]byte{8, 0, 0, 0})
			case 9: // GetObstruction → not active
				_, _ = conn.Write([]byte{9, 0, 0, 0})
			default:
				// Write commands – no reply needed
			}
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }
}

// TestMain initialises elevio with a mock server before any test in this
// package is run.
func TestMain(m *testing.M) {
	// Use a fixed port in the test-only path to keep it simple; start a
	// throw-away listener just to get an ephemeral port and address.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: listen: %v\n", err)
		os.Exit(1)
	}
	addr := ln.Addr().String()

	// Serve in the background
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveConn(conn)
		}
	}()

	elevio.Init(addr, NumFloors)
	os.Exit(m.Run())
}

func serveConn(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 4)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			return
		}
		switch buf[0] {
		case 6:
			_, _ = conn.Write([]byte{6, 0, 0, 0})
		case 7:
			_, _ = conn.Write([]byte{7, 1, 0, 0})
		case 8:
			_, _ = conn.Write([]byte{8, 0, 0, 0})
		case 9:
			_, _ = conn.Write([]byte{9, 0, 0, 0})
		}
	}
}
