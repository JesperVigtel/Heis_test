// Package elevio provides the interface to the elevator hardware simulator.
// It communicates via TCP to the elevator server (default port 15657).
package elevio

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// NumFloors is the number of floors the elevator operates on.
const NumFloors = 4

// Button types
type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown ButtonType = 1
	BT_Cab      ButtonType = 2
)

// MotorDirection represents the direction the elevator motor is running.
type MotorDirection int

const (
	MD_Up   MotorDirection = 1
	MD_Down MotorDirection = -1
	MD_Stop MotorDirection = 0
)

// ButtonEvent is emitted when a button is pressed.
type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

var (
	_initialized bool
	_numFloors   int
	_mtx         sync.Mutex
	_conn        net.Conn
)

// Init initialises the elevator IO library and connects to the elevator server.
func Init(addr string, numFloors int) {
	if _initialized {
		fmt.Println("elevio: already initialized")
		return
	}
	_numFloors = numFloors
	var err error
	for {
		_conn, err = net.DialTimeout("tcp", addr, 5*time.Second)
		if err == nil {
			break
		}
		fmt.Printf("elevio: failed to connect to %s: %v – retrying in 1s\n", addr, err)
		time.Sleep(1 * time.Second)
	}
	_initialized = true
}

func write(msg []byte) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_, err := _conn.Write(msg)
	if err != nil {
		fmt.Printf("elevio: write error: %v\n", err)
	}
}

// writeRead sends a read-command and reads the 4-byte reply atomically under
// the mutex, preventing concurrent polling goroutines from interleaving their
// request/response pairs.
func writeRead(msg []byte, buf []byte) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_, err := _conn.Write(msg)
	if err != nil {
		fmt.Printf("elevio: write error: %v\n", err)
	}
	_, err = _conn.Read(buf)
	if err != nil {
		fmt.Printf("elevio: read error: %v\n", err)
	}
}

// SetMotorDirection commands the motor to move up, down, or stop.
func SetMotorDirection(dir MotorDirection) {
	write([]byte{1, byte(dir), 0, 0})
}

// SetButtonLamp turns a button lamp on or off.
func SetButtonLamp(button ButtonType, floor int, value bool) {
	var v byte
	if value {
		v = 1
	}
	write([]byte{2, byte(button), byte(floor), v})
}

// SetFloorIndicator sets the floor indicator display.
func SetFloorIndicator(floor int) {
	write([]byte{3, byte(floor), 0, 0})
}

// SetDoorOpenLamp controls the door open lamp.
func SetDoorOpenLamp(value bool) {
	var v byte
	if value {
		v = 1
	}
	write([]byte{4, v, 0, 0})
}

// SetStopLamp controls the stop button lamp.
func SetStopLamp(value bool) {
	var v byte
	if value {
		v = 1
	}
	write([]byte{5, v, 0, 0})
}

// GetFloor returns the current floor sensor value (-1 if between floors).
func GetFloor() int {
	var buf [4]byte
	writeRead([]byte{7, 0, 0, 0}, buf[:])
	if buf[1] == 0 {
		return -1
	}
	return int(buf[2])
}

// GetButton returns true if the given button is currently pressed.
func GetButton(button ButtonType, floor int) bool {
	var buf [4]byte
	writeRead([]byte{6, byte(button), byte(floor), 0}, buf[:])
	return buf[1] != 0
}

// GetObstruction returns true if the obstruction switch is active.
func GetObstruction() bool {
	var buf [4]byte
	writeRead([]byte{9, 0, 0, 0}, buf[:])
	return buf[1] != 0
}

// GetStop returns true if the stop button is pressed.
func GetStop() bool {
	var buf [4]byte
	writeRead([]byte{8, 0, 0, 0}, buf[:])
	return buf[1] != 0
}

// PollButtons polls all buttons and sends events to ch when a button transitions
// from not pressed to pressed.
func PollButtons(ch chan<- ButtonEvent) {
	var prev [NumFloors][3]bool
	for {
		time.Sleep(25 * time.Millisecond)
		for f := 0; f < _numFloors; f++ {
			for b := ButtonType(0); b < 3; b++ {
				// Skip hall buttons that don't exist at the extremes
				if (b == BT_HallUp && f == _numFloors-1) ||
					(b == BT_HallDown && f == 0) {
					continue
				}
				v := GetButton(b, f)
				if v && !prev[f][b] {
					ch <- ButtonEvent{Floor: f, Button: b}
				}
				prev[f][b] = v
			}
		}
	}
}

// PollFloorSensor polls the floor sensor and sends the floor number to ch
// whenever the elevator arrives at a new floor.
func PollFloorSensor(ch chan<- int) {
	prev := -2
	for {
		time.Sleep(25 * time.Millisecond)
		v := GetFloor()
		if v != prev && v != -1 {
			ch <- v
			prev = v
		}
	}
}

// PollObstructionSwitch polls the obstruction switch.
func PollObstructionSwitch(ch chan<- bool) {
	prev := false
	for {
		time.Sleep(25 * time.Millisecond)
		v := GetObstruction()
		if v != prev {
			ch <- v
			prev = v
		}
	}
}

// PollStopButton polls the stop button.
func PollStopButton(ch chan<- bool) {
	prev := false
	for {
		time.Sleep(25 * time.Millisecond)
		v := GetStop()
		if v != prev {
			ch <- v
			prev = v
		}
	}
}
