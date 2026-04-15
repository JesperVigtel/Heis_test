package cluster

import (
	"cmp"
	"slices"
	"time"

	"new_heis/internal/model"
)

type peer struct {
	state model.SoftState
	seen  time.Time
}

type PeerTracker struct {
	timeout time.Duration
	peers   map[int]peer
}

func NewPeerTracker(timeout time.Duration) *PeerTracker {
	return &PeerTracker{timeout: timeout, peers: map[int]peer{}}
}

func (t *PeerTracker) Observe(state model.SoftState, now time.Time) bool {
	current, ok := t.peers[state.ID]
	if ok && state.BootID < current.state.BootID {
		return false
	}
	changed := !ok || current.state != state
	t.peers[state.ID] = peer{state: state, seen: now}
	return changed
}

func (t *PeerTracker) Expire(now time.Time) bool {
	changed := false
	for id, peer := range t.peers {
		if now.Sub(peer.seen) > t.timeout {
			delete(t.peers, id)
			changed = true
		}
	}
	return changed
}

func (t *PeerTracker) Connected() bool {
	return len(t.peers) > 0
}

func (t *PeerTracker) LiveStates(self model.SoftState) []model.SoftState {
	out := make([]model.SoftState, 1, len(t.peers)+1)
	out[0] = self
	for _, peer := range t.peers {
		out = append(out, peer.state)
	}
	slices.SortFunc(out, func(a, b model.SoftState) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return out
}
