// fsm.go implements the event-driven FSM for a single elevator.
package elevator

import (
	"fmt"

	"github.com/JesperVigtel/Heis_test/elevio"
)

// SetAllLights refreshes all button lights based on the elevator's request
// matrix. Hall lights are shared so callers may pass in the combined hall
// request state from the network instead.
func SetAllLights(e Elevator) {
	for floor := 0; floor < NumFloors; floor++ {
		for btn := 0; btn < NumButtons; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, e.Requests[floor][btn])
		}
	}
}

// OnInitBetweenFloors handles startup when the elevator is between floors.
// It moves the elevator downward until it hits a floor sensor.
func OnInitBetweenFloors(e *Elevator) {
	elevio.SetMotorDirection(elevio.MD_Down)
	e.Dir = DirDown
	e.Behaviour = Moving
}

// OnFloorArrival handles a floor arrival event. It returns the updated
// elevator and whether the door timer should be (re)started.
func OnFloorArrival(e Elevator, newFloor int) (Elevator, bool) {
	e.Floor = newFloor
	elevio.SetFloorIndicator(e.Floor)

	switch e.Behaviour {
	case Moving:
		if ShouldStop(e) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			e = ClearAtCurrentFloor(e)
			e.Behaviour = DoorOpen
			return e, true // restart door timer
		}
	default:
	}
	return e, false
}

// OnDoorTimeout handles the door timer expiry. Returns the updated elevator
// and whether the door timer should be restarted (e.g. direction change).
func OnDoorTimeout(e Elevator) (Elevator, bool) {
	if e.Obstructed {
		// Keep door open; caller will restart timer
		return e, true
	}
	if e.Behaviour != DoorOpen {
		return e, false
	}

	dir, behaviour := ChooseDirection(e)
	e.Dir = dir
	e.Behaviour = behaviour

	switch behaviour {
	case DoorOpen:
		// Change of direction announced by keeping door open another cycle
		elevio.SetDoorOpenLamp(true)
		e = ClearAtCurrentFloor(e)
		return e, true
	case Moving:
		elevio.SetDoorOpenLamp(false)
		elevio.SetMotorDirection(dir)
		return e, false
	case Idle:
		elevio.SetDoorOpenLamp(false)
		elevio.SetMotorDirection(elevio.MD_Stop)
		return e, false
	}
	return e, false
}

// OnRequestButtonPress handles a button press event.
// It returns the updated elevator and whether the door timer should start.
func OnRequestButtonPress(e Elevator, btnFloor int, btnType elevio.ButtonType) (Elevator, bool) {
	fmt.Printf("FSM: button %v floor %d\n", btnType, btnFloor)

	e.Requests[btnFloor][int(btnType)] = true

	switch e.Behaviour {
	case DoorOpen:
		if e.Floor == btnFloor {
			e = ClearAtCurrentFloor(e)
			return e, true // restart timer
		}
	case Idle:
		if e.Floor == btnFloor {
			elevio.SetDoorOpenLamp(true)
			e = ClearAtCurrentFloor(e)
			e.Behaviour = DoorOpen
			return e, true
		}
		dir, behaviour := ChooseDirection(e)
		e.Dir = dir
		e.Behaviour = behaviour
		switch behaviour {
		case Moving:
			elevio.SetMotorDirection(dir)
		case DoorOpen:
			elevio.SetDoorOpenLamp(true)
			e = ClearAtCurrentFloor(e)
		}
	case Moving:
		// Nothing special – the request will be picked up when we arrive
	}
	return e, false
}
