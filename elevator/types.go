// Package elevator defines the types and FSM logic for a single elevator.
package elevator

import (
	"github.com/JesperVigtel/Heis_test/elevio"
)

// NumFloors is the number of floors.
const NumFloors = elevio.NumFloors

// NumButtons is the number of button types (HallUp, HallDown, Cab).
const NumButtons = 3

// State represents the FSM state of the elevator.
type State int

const (
	Idle     State = iota
	DoorOpen       // door is open, waiting before closing
	Moving         // elevator is moving
)

// Behaviour is an alias kept for human-readable state strings.
type Behaviour = State

// Dir represents the last/current direction of travel.
type Dir = elevio.MotorDirection

const (
	DirUp   = elevio.MD_Up
	DirDown = elevio.MD_Down
	DirStop = elevio.MD_Stop
)

// Elevator holds the complete state of one elevator.
type Elevator struct {
	Floor     int
	Dir       Dir
	Requests  [NumFloors][NumButtons]bool
	Behaviour State
	Config    Config
	Obstructed bool
}

// Config holds runtime configuration.
type Config struct {
	DoorOpenDuration float64 // seconds
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{DoorOpenDuration: 3.0}
}

// UninitializedElevator returns a zero-value Elevator ready for use.
func UninitializedElevator() Elevator {
	return Elevator{
		Floor:     -1,
		Dir:       DirStop,
		Behaviour: Idle,
		Config:    DefaultConfig(),
	}
}
