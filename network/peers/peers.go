// Package peers provides peer discovery via UDP heartbeats.
// Each node broadcasts its ID periodically; if a node stops broadcasting it
// is removed from the peer list after a timeout.
package peers

import (
	"fmt"
	"net"
	"sort"
	"time"
)

// PeerUpdate is sent on the update channel whenever the peer list changes.
type PeerUpdate struct {
	Peers []string // sorted list of peer IDs currently alive
	New   string   // newly appeared peer (empty if none)
	Lost  []string // peers that have disappeared
}

const (
	heartbeatInterval = 15 * time.Millisecond
	peerTimeout       = 500 * time.Millisecond
)

// Transmitter broadcasts our ID on the given UDP port so that other nodes can
// discover us. enable is an optional channel to pause/resume transmission.
func Transmitter(port int, id string, enable <-chan bool) {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))
	if err != nil {
		panic(fmt.Sprintf("peers Transmitter: %v", err))
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		panic(fmt.Sprintf("peers Transmitter: %v", err))
	}
	defer conn.Close()

	enabled := true
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case enabled = <-enable:
		case <-ticker.C:
			if enabled {
				_, _ = conn.Write([]byte(id))
			}
		}
	}
}

// Receiver listens for heartbeats and sends PeerUpdate messages on updates
// whenever the peer list changes.
func Receiver(port int, updates chan<- PeerUpdate) {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("peers Receiver: %v", err))
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		panic(fmt.Sprintf("peers Receiver: %v", err))
	}
	defer conn.Close()

	peerLastSeen := make(map[string]time.Time)
	checkTimeout := time.NewTicker(peerTimeout / 2)
	defer checkTimeout.Stop()

	var buf [1024]byte
	for {
		// Non-blocking check of incoming heartbeats
		conn.SetReadDeadline(time.Now().Add(heartbeatInterval))
		n, _, err := conn.ReadFromUDP(buf[:])
		if err == nil && n > 0 {
			id := string(buf[:n])
			_, existed := peerLastSeen[id]
			peerLastSeen[id] = time.Now()
			if !existed {
				// New peer appeared
				updates <- buildUpdate(peerLastSeen, id, nil)
			}
		}

		// Check for timed-out peers
		select {
		case <-checkTimeout.C:
			lost := make([]string, 0, len(peerLastSeen))
			now := time.Now()
			for id, t := range peerLastSeen {
				if now.Sub(t) > peerTimeout {
					lost = append(lost, id)
					delete(peerLastSeen, id)
				}
			}
			if len(lost) > 0 {
				updates <- buildUpdate(peerLastSeen, "", lost)
			}
		default:
		}
	}
}

func buildUpdate(peerLastSeen map[string]time.Time, newPeer string, lost []string) PeerUpdate {
	peers := make([]string, 0, len(peerLastSeen))
	for id := range peerLastSeen {
		peers = append(peers, id)
	}
	sort.Strings(peers)
	return PeerUpdate{
		Peers: peers,
		New:   newPeer,
		Lost:  lost,
	}
}
