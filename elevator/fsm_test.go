package elevator

import (
	"testing"

	"github.com/JesperVigtel/Heis_test/elevio"
)

// ---- OnFloorArrival ----

func TestOnFloorArrival_MovingAndShouldStop(t *testing.T) {
	e := makeElevator(0, DirUp, Moving)
	e.Requests[1][int(elevio.BT_Cab)] = true

	e2, startTimer := OnFloorArrival(e, 1)
	if !startTimer {
		t.Error("expected door timer to start on floor arrival where stop is needed")
	}
	if e2.Behaviour != DoorOpen {
		t.Errorf("expected DoorOpen after stopping, got %v", e2.Behaviour)
	}
	if e2.Floor != 1 {
		t.Errorf("expected floor 1, got %d", e2.Floor)
	}
	if e2.Requests[1][int(elevio.BT_Cab)] {
		t.Error("expected cab request cleared after stop")
	}
}

func TestOnFloorArrival_MovingAndShouldNotStop(t *testing.T) {
	e := makeElevator(0, DirUp, Moving)
	e.Requests[3][int(elevio.BT_Cab)] = true

	// Arrive at floor 1 – should not stop because target is floor 3
	e2, startTimer := OnFloorArrival(e, 1)
	if startTimer {
		t.Error("expected no door timer when not stopping")
	}
	if e2.Behaviour != Moving {
		t.Errorf("expected still Moving, got %v", e2.Behaviour)
	}
	if e2.Floor != 1 {
		t.Errorf("expected floor updated to 1, got %d", e2.Floor)
	}
}

func TestOnFloorArrival_NotMoving(t *testing.T) {
	// While Idle or DoorOpen, floor arrivals should be ignored (no-op)
	e := makeElevator(0, DirStop, Idle)
	e2, startTimer := OnFloorArrival(e, 2)
	if startTimer {
		t.Error("expected no timer for non-moving state")
	}
	if e2.Floor != 2 {
		t.Errorf("expected floor updated to 2, got %d", e2.Floor)
	}
	if e2.Behaviour != Idle {
		t.Errorf("expected Idle behaviour preserved, got %v", e2.Behaviour)
	}
}

// ---- OnDoorTimeout ----

func TestOnDoorTimeout_ObstructedKeepsDoorOpen(t *testing.T) {
	e := makeElevator(1, DirStop, DoorOpen)
	e.Obstructed = true
	e2, startTimer := OnDoorTimeout(e)
	if !startTimer {
		t.Error("expected door timer restart when obstructed")
	}
	if e2.Behaviour != DoorOpen {
		t.Errorf("expected DoorOpen while obstructed, got %v", e2.Behaviour)
	}
}

func TestOnDoorTimeout_NotDoorOpenState(t *testing.T) {
	e := makeElevator(1, DirStop, Idle)
	e2, startTimer := OnDoorTimeout(e)
	if startTimer {
		t.Error("expected no timer restart when not in DoorOpen state")
	}
	if e2.Behaviour != Idle {
		t.Errorf("expected Idle preserved, got %v", e2.Behaviour)
	}
}

func TestOnDoorTimeout_NoRemainingRequests_GoesIdle(t *testing.T) {
	e := makeElevator(2, DirStop, DoorOpen)
	// No requests → should go Idle
	e2, startTimer := OnDoorTimeout(e)
	if startTimer {
		t.Error("expected no timer restart when going idle")
	}
	if e2.Behaviour != Idle {
		t.Errorf("expected Idle after timeout with no requests, got %v", e2.Behaviour)
	}
}

func TestOnDoorTimeout_PendingRequestAbove_StartMoving(t *testing.T) {
	e := makeElevator(0, DirStop, DoorOpen)
	e.Requests[3][int(elevio.BT_Cab)] = true
	e2, startTimer := OnDoorTimeout(e)
	if startTimer {
		t.Error("expected no door timer when starting to move")
	}
	if e2.Behaviour != Moving {
		t.Errorf("expected Moving after timeout with request above, got %v", e2.Behaviour)
	}
	if e2.Dir != DirUp {
		t.Errorf("expected DirUp, got %v", e2.Dir)
	}
}

func TestOnDoorTimeout_PendingRequestBelow_StartMoving(t *testing.T) {
	e := makeElevator(3, DirStop, DoorOpen)
	e.Requests[0][int(elevio.BT_Cab)] = true
	e2, startTimer := OnDoorTimeout(e)
	if startTimer {
		t.Error("expected no door timer when starting to move")
	}
	if e2.Behaviour != Moving {
		t.Errorf("expected Moving after timeout with request below, got %v", e2.Behaviour)
	}
	if e2.Dir != DirDown {
		t.Errorf("expected DirDown, got %v", e2.Dir)
	}
}

// ---- OnRequestButtonPress ----

func TestOnRequestButtonPress_CabOnCurrentFloorWhileIdle(t *testing.T) {
	e := makeElevator(2, DirStop, Idle)
	e2, startTimer := OnRequestButtonPress(e, 2, elevio.BT_Cab)
	if !startTimer {
		t.Error("expected door timer to start for cab request on current floor")
	}
	if e2.Behaviour != DoorOpen {
		t.Errorf("expected DoorOpen, got %v", e2.Behaviour)
	}
	if e2.Requests[2][int(elevio.BT_Cab)] {
		t.Error("expected cab request immediately cleared at current floor")
	}
}

func TestOnRequestButtonPress_CabOnOtherFloorWhileIdle(t *testing.T) {
	e := makeElevator(0, DirStop, Idle)
	e2, startTimer := OnRequestButtonPress(e, 3, elevio.BT_Cab)
	if startTimer {
		t.Error("expected no door timer for cab request on a different floor while idle")
	}
	if !e2.Requests[3][int(elevio.BT_Cab)] {
		t.Error("expected cab request registered for floor 3")
	}
	if e2.Behaviour != Moving {
		t.Errorf("expected Moving towards floor 3, got %v", e2.Behaviour)
	}
	if e2.Dir != DirUp {
		t.Errorf("expected DirUp, got %v", e2.Dir)
	}
}

func TestOnRequestButtonPress_WhileDoorOpenOnSameFloor(t *testing.T) {
	e := makeElevator(1, DirStop, DoorOpen)
	e2, startTimer := OnRequestButtonPress(e, 1, elevio.BT_Cab)
	if !startTimer {
		t.Error("expected door timer restart for request on current floor while DoorOpen")
	}
	// The cab request at the current floor should be cleared immediately.
	if e2.Requests[1][int(elevio.BT_Cab)] {
		t.Error("expected cab request cleared at current floor while DoorOpen")
	}
}

func TestOnRequestButtonPress_WhileMoving(t *testing.T) {
	e := makeElevator(0, DirUp, Moving)
	e2, startTimer := OnRequestButtonPress(e, 3, elevio.BT_Cab)
	if startTimer {
		t.Error("expected no door timer while moving")
	}
	if !e2.Requests[3][int(elevio.BT_Cab)] {
		t.Error("expected request registered while moving")
	}
	if e2.Behaviour != Moving {
		t.Errorf("expected to remain Moving, got %v", e2.Behaviour)
	}
}

// ---- OnInitBetweenFloors ----

func TestOnInitBetweenFloors(t *testing.T) {
	e := UninitializedElevator()
	OnInitBetweenFloors(&e)
	if e.Dir != DirDown {
		t.Errorf("expected DirDown on init between floors, got %v", e.Dir)
	}
	if e.Behaviour != Moving {
		t.Errorf("expected Moving on init between floors, got %v", e.Behaviour)
	}
}
