package elevator

import "github.com/JesperVigtel/Heis_test/elevio"

// requestsAbove returns true if there are any requests on floors above current.
func requestsAbove(e Elevator) bool {
	for f := e.Floor + 1; f < NumFloors; f++ {
		for b := 0; b < NumButtons; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

// requestsBelow returns true if there are any requests on floors below current.
func requestsBelow(e Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for b := 0; b < NumButtons; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

// requestsHere returns true if there are any requests on the current floor.
func requestsHere(e Elevator) bool {
	for b := 0; b < NumButtons; b++ {
		if e.Requests[e.Floor][b] {
			return true
		}
	}
	return false
}

// ChooseDirection returns the direction the elevator should move, and what
// behaviour it should enter (Moving or Idle).
func ChooseDirection(e Elevator) (Dir, State) {
	switch e.Dir {
	case DirUp:
		if requestsAbove(e) {
			return DirUp, Moving
		}
		if requestsHere(e) {
			return DirDown, DoorOpen
		}
		if requestsBelow(e) {
			return DirDown, Moving
		}
		return DirStop, Idle
	case DirDown:
		if requestsBelow(e) {
			return DirDown, Moving
		}
		if requestsHere(e) {
			return DirUp, DoorOpen
		}
		if requestsAbove(e) {
			return DirUp, Moving
		}
		return DirStop, Idle
	case DirStop:
		if requestsHere(e) {
			return DirStop, DoorOpen
		}
		if requestsAbove(e) {
			return DirUp, Moving
		}
		if requestsBelow(e) {
			return DirDown, Moving
		}
		return DirStop, Idle
	default:
		return DirStop, Idle
	}
}

// ShouldStop returns true if the elevator should stop at the current floor.
func ShouldStop(e Elevator) bool {
	switch e.Dir {
	case DirDown:
		return e.Requests[e.Floor][int(elevio.BT_HallDown)] ||
			e.Requests[e.Floor][int(elevio.BT_Cab)] ||
			!requestsBelow(e)
	case DirUp:
		return e.Requests[e.Floor][int(elevio.BT_HallUp)] ||
			e.Requests[e.Floor][int(elevio.BT_Cab)] ||
			!requestsAbove(e)
	case DirStop:
		return true
	default:
		return true
	}
}

// ClearAtCurrentFloor clears the appropriate requests at the current floor
// based on the direction of travel, and returns the updated elevator.
func ClearAtCurrentFloor(e Elevator) Elevator {
	// Always clear cab request
	e.Requests[e.Floor][int(elevio.BT_Cab)] = false

	switch e.Dir {
	case DirUp:
		if !requestsAbove(e) && !e.Requests[e.Floor][int(elevio.BT_HallUp)] {
			e.Requests[e.Floor][int(elevio.BT_HallDown)] = false
		}
		e.Requests[e.Floor][int(elevio.BT_HallUp)] = false
	case DirDown:
		if !requestsBelow(e) && !e.Requests[e.Floor][int(elevio.BT_HallDown)] {
			e.Requests[e.Floor][int(elevio.BT_HallUp)] = false
		}
		e.Requests[e.Floor][int(elevio.BT_HallDown)] = false
	case DirStop:
		e.Requests[e.Floor][int(elevio.BT_HallUp)] = false
		e.Requests[e.Floor][int(elevio.BT_HallDown)] = false
	}
	return e
}
