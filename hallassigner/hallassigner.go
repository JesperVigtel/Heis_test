// Package hallassigner implements the hall request assignment algorithm.
// It uses a cost function to assign pending hall requests to elevators.
package hallassigner

import (
	"github.com/JesperVigtel/Heis_test/elevator"
	"github.com/JesperVigtel/Heis_test/elevio"
)

// HallRequestState describes whether a hall request is active and whether it
// has been assigned to an elevator.
type HallRequestState int

const (
	None     HallRequestState = iota
	Unassigned
	Assigned
)

// HallRequests holds the assignment state for hall calls [floor][direction].
// direction: 0 = HallUp, 1 = HallDown.
type HallRequests = [elevio.NumFloors][2]HallRequestState

// ElevatorState is the view of an elevator state used for cost calculation.
type ElevatorState struct {
	Behaviour elevator.State
	Floor     int
	Direction elevator.Dir
	// CabRequests holds cab button state for the elevator.
	CabRequests [elevio.NumFloors]bool
}

// Assign computes a new hall request assignment given the current peer states
// and the pending hall requests. It returns a map from elevator ID to the full
// request matrix (hall + cab) for that elevator.
//
// The assignment uses a simple travel-time cost function.
func Assign(
	hallRequests [elevio.NumFloors][2]bool,
	elevators map[string]ElevatorState,
) map[string][elevio.NumFloors][elevator.NumButtons]bool {
	result := make(map[string][elevio.NumFloors][elevator.NumButtons]bool)

	// Initialise result matrices with cab requests
	for id, e := range elevators {
		var matrix [elevio.NumFloors][elevator.NumButtons]bool
		for f := 0; f < elevio.NumFloors; f++ {
			matrix[f][int(elevio.BT_Cab)] = e.CabRequests[f]
		}
		result[id] = matrix
	}

	if len(elevators) == 0 {
		return result
	}

	// Assign each hall request to the elevator with the lowest cost.
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if !hallRequests[floor][btn] {
				continue
			}
			bestID := ""
			bestCost := int(^uint(0) >> 1) // MaxInt
			for id, e := range elevators {
				c := cost(e, floor, elevio.ButtonType(btn))
				if c < bestCost {
					bestCost = c
					bestID = id
				}
			}
			if bestID != "" {
				m := result[bestID]
				m[floor][btn] = true
				result[bestID] = m
			}
		}
	}
	return result
}

// btnToDir maps a hall button type to the corresponding motor direction.
// BT_HallUp → MD_Up, BT_HallDown → MD_Down.
func btnToDir(btn elevio.ButtonType) elevio.MotorDirection {
	if btn == elevio.BT_HallUp {
		return elevio.MD_Up
	}
	return elevio.MD_Down
}

// cost estimates the time (in simulated floor-travel units) for the elevator e
// to service a hall request at targetFloor with button type btn.
func cost(e ElevatorState, targetFloor int, btn elevio.ButtonType) int {
	const (
		travelTime = 2 // floors of distance cost weight (travel 1 floor ≈ 2 units)
		doorTime   = 1 // cost for already having door open
	)

	d := e.Floor
	dir := e.Direction
	duration := 0

	if e.Behaviour == elevator.DoorOpen {
		duration += doorTime
	}

	for {
		if d == targetFloor {
			requestedDir := btnToDir(btn)
			switch {
			case dir == requestedDir:
				return duration
			case dir == elevio.MD_Stop:
				return duration
			}
		}

		switch {
		case dir == elevio.MD_Up:
			if d < elevio.NumFloors-1 {
				d++
			} else {
				dir = elevio.MD_Down
			}
		case dir == elevio.MD_Down:
			if d > 0 {
				d--
			} else {
				dir = elevio.MD_Up
			}
		default:
			if targetFloor > d {
				dir = elevio.MD_Up
			} else {
				dir = elevio.MD_Down
			}
		}
		duration += travelTime

		if duration > 100 {
			break
		}
	}
	return duration
}
