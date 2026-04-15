package model

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"time"
)

const DefaultFloors = 4

type Direction int

const (
	DirectionDown Direction = -1
	DirectionStop Direction = 0
	DirectionUp   Direction = 1
)

func (d Direction) String() string {
	switch d {
	case DirectionUp:
		return "up"
	case DirectionDown:
		return "down"
	default:
		return "stop"
	}
}

type Behavior string

const (
	BehaviorIdle     Behavior = "idle"
	BehaviorMoving   Behavior = "moving"
	BehaviorDoorOpen Behavior = "doorOpen"
	BehaviorStuck    Behavior = "stuck"
)

type Button int

const (
	ButtonHallUp Button = iota
	ButtonHallDown
	ButtonCab
)

func (b Button) String() string {
	switch b {
	case ButtonHallUp:
		return "hall-up"
	case ButtonHallDown:
		return "hall-down"
	default:
		return "cab"
	}
}

func (b Button) Direction() Direction {
	switch b {
	case ButtonHallUp:
		return DirectionUp
	case ButtonHallDown:
		return DirectionDown
	default:
		return DirectionStop
	}
}

type RequestKind string

const (
	RequestKindHall RequestKind = "hall"
	RequestKindCab  RequestKind = "cab"
)

type RequestKey struct {
	Kind      RequestKind `json:"kind"`
	Floor     int         `json:"floor"`
	Direction Direction   `json:"direction,omitempty"`
	OwnerID   int         `json:"ownerID,omitempty"`
}

func HallKey(floor int, dir Direction) RequestKey {
	return RequestKey{Kind: RequestKindHall, Floor: floor, Direction: dir}
}

func CabKey(owner, floor int) RequestKey {
	return RequestKey{Kind: RequestKindCab, Floor: floor, OwnerID: owner}
}

type Version struct {
	Counter  uint64 `json:"counter"`
	WriterID int    `json:"writerID"`
	BootID   string `json:"bootID"`
}

func CompareVersion(a, b Version) int {
	if a.WriterID == b.WriterID && a.BootID != b.BootID {
		return cmp.Compare(a.BootID, b.BootID)
	}
	if d := cmp.Compare(a.Counter, b.Counter); d != 0 {
		return d
	}
	if d := cmp.Compare(a.WriterID, b.WriterID); d != 0 {
		return d
	}
	return cmp.Compare(a.BootID, b.BootID)
}

type RequestCell struct {
	Active  bool    `json:"active"`
	Version Version `json:"version"`
}

type RequestEntry struct {
	Key  RequestKey  `json:"key"`
	Cell RequestCell `json:"cell"`
}

type RequestTable map[RequestKey]RequestCell

func NewRequestTable() RequestTable {
	return RequestTable{}
}

func (t RequestTable) Cell(key RequestKey) RequestCell {
	return t[key]
}

func (t RequestTable) Active(key RequestKey) bool {
	return t[key].Active
}

func (t RequestTable) Set(key RequestKey, cell RequestCell) {
	t[key] = cell
}

func (t RequestTable) Entries() []RequestEntry {
	return t.entries(false)
}

func (t RequestTable) ActiveEntries() []RequestEntry {
	return t.entries(true)
}

func (t RequestTable) MaxCounter() uint64 {
	var max uint64
	for _, cell := range t {
		if cell.Version.Counter > max {
			max = cell.Version.Counter
		}
	}
	return max
}

func (t RequestTable) entries(activeOnly bool) []RequestEntry {
	out := make([]RequestEntry, 0, len(t))
	for key, cell := range t {
		if activeOnly && !cell.Active {
			continue
		}
		out = append(out, RequestEntry{Key: key, Cell: cell})
	}
	slices.SortFunc(out, func(a, b RequestEntry) int {
		return compareKey(a.Key, b.Key)
	})
	return out
}

type Clock struct {
	value  uint64
	writer int
	boot   string
}

func NewClock(writer int, boot string) Clock {
	return Clock{writer: writer, boot: boot}
}

func (c *Clock) Next() Version {
	c.value++
	return Version{Counter: c.value, WriterID: c.writer, BootID: c.boot}
}

func (c *Clock) Observe(v Version) {
	if v.Counter > c.value {
		c.value = v.Counter
	}
}

func (c *Clock) Seed(n uint64) {
	if n > c.value {
		c.value = n
	}
}

func NewBootID(now time.Time) string {
	return fmt.Sprintf("%020d-%06d", now.UTC().UnixNano(), os.Getpid())
}

type SoftState struct {
	ID          int       `json:"id"`
	BootID      string    `json:"bootID"`
	Floor       int       `json:"floor"`
	Direction   Direction `json:"direction"`
	Behavior    Behavior  `json:"behavior"`
	DoorOpen    bool      `json:"doorOpen"`
	Obstruction bool      `json:"obstruction"`
}

type Snapshot struct {
	Soft     SoftState      `json:"soft"`
	Requests []RequestEntry `json:"requests"`
}

func compareKey(a, b RequestKey) int {
	if d := cmp.Compare(string(a.Kind), string(b.Kind)); d != 0 {
		return d
	}
	if d := cmp.Compare(a.OwnerID, b.OwnerID); d != 0 {
		return d
	}
	if d := cmp.Compare(a.Floor, b.Floor); d != 0 {
		return d
	}
	return cmp.Compare(int(a.Direction), int(b.Direction))
}
