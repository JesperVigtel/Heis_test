package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"new_heis/internal/model"
)

type Config struct {
	Port     int
	Interval time.Duration
	Targets  []string
}

type Node struct {
	conn net.PacketConn
	in   chan model.Snapshot
	mu   sync.RWMutex
	last model.Snapshot
}

func Start(ctx context.Context, cfg Config) (*Node, error) {
	conn, err := listenBroadcastUDP(cfg.Port)
	if err != nil {
		return nil, err
	}
	n := &Node{
		conn: conn,
		in:   make(chan model.Snapshot, 64),
	}
	addrs := targets(cfg)
	go n.recv(ctx)
	go n.send(ctx, cfg.Interval, addrs)
	return n, nil
}

func (n *Node) Incoming() <-chan model.Snapshot { return n.in }

func (n *Node) SetLocal(snapshot model.Snapshot) {
	n.mu.Lock()
	n.last = snapshot
	n.mu.Unlock()
}

func (n *Node) Close() error { return n.conn.Close() }

func (n *Node) send(ctx context.Context, interval time.Duration, addrs []*net.UDPAddr) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.mu.RLock()
			payload, err := json.Marshal(n.last)
			n.mu.RUnlock()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				_, _ = n.conn.WriteTo(payload, addr)
			}
		}
	}
}

func (n *Node) recv(ctx context.Context) {
	buf := make([]byte, 64<<10)
	for {
		_ = n.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		size, _, err := n.conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			if ctx.Err() != nil {
				return
			}
			continue
		}

		var snap model.Snapshot
		if json.Unmarshal(buf[:size], &snap) != nil {
			continue
		}
		select {
		case n.in <- snap:
		case <-ctx.Done():
			return
		default:
		}
	}
}

func targets(cfg Config) []*net.UDPAddr {
	raw := cfg.Targets
	if len(raw) == 0 {
		raw = []string{fmt.Sprintf("255.255.255.255:%d", cfg.Port)}
	}
	out := make([]*net.UDPAddr, 0, len(raw))
	for _, target := range raw {
		if addr, err := net.ResolveUDPAddr("udp4", target); err == nil {
			out = append(out, addr)
		}
	}
	return out
}
