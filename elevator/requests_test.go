package elevator

import (
	"testing"

	"github.com/JesperVigtel/Heis_test/elevio"
)

// makeElevator constructs an Elevator for use in tests without network I/O.
func makeElevator(floor int, dir Dir, state State) Elevator {
	e := UninitializedElevator()
	e.Floor = floor
	e.Dir = dir
	e.Behaviour = state
	return e
}

// setRequests sets requests at the given (floor, button) pairs.
func setRequests(e *Elevator, reqs [][2]int) {
	for _, r := range reqs {
		e.Requests[r[0]][r[1]] = true
	}
}

// ---- requestsAbove / requestsBelow / requestsHere ----

func TestRequestsAbove(t *testing.T) {
	e := makeElevator(1, DirUp, Moving)
	if requestsAbove(e) {
		t.Error("expected no requests above")
	}
	e.Requests[3][int(elevio.BT_Cab)] = true
	if !requestsAbove(e) {
		t.Error("expected requests above")
	}
}

func TestRequestsBelow(t *testing.T) {
	e := makeElevator(2, DirDown, Moving)
	if requestsBelow(e) {
		t.Error("expected no requests below")
	}
	e.Requests[0][int(elevio.BT_Cab)] = true
	if !requestsBelow(e) {
		t.Error("expected requests below")
	}
}

func TestRequestsHere(t *testing.T) {
	e := makeElevator(1, DirStop, Idle)
	if requestsHere(e) {
		t.Error("expected no requests here")
	}
	e.Requests[1][int(elevio.BT_Cab)] = true
	if !requestsHere(e) {
		t.Error("expected request here")
	}
}

// ---- ShouldStop ----

func TestShouldStop_CabRequest(t *testing.T) {
	for _, dir := range []Dir{DirUp, DirDown} {
		e := makeElevator(2, dir, Moving)
		e.Requests[2][int(elevio.BT_Cab)] = true
		if !ShouldStop(e) {
			t.Errorf("expected stop on cab request while moving %v", dir)
		}
	}
}

func TestShouldStop_HallUpWhileGoingUp(t *testing.T) {
	e := makeElevator(1, DirUp, Moving)
	e.Requests[1][int(elevio.BT_HallUp)] = true
	e.Requests[3][int(elevio.BT_Cab)] = true // still something above
	if !ShouldStop(e) {
		t.Error("expected stop on hall-up request when going up")
	}
}

func TestShouldStop_HallDownWhileGoingDown(t *testing.T) {
	e := makeElevator(2, DirDown, Moving)
	e.Requests[2][int(elevio.BT_HallDown)] = true
	e.Requests[0][int(elevio.BT_Cab)] = true // still something below
	if !ShouldStop(e) {
		t.Error("expected stop on hall-down request when going down")
	}
}

func TestShouldStop_NoMoreRequestsAbove(t *testing.T) {
	e := makeElevator(2, DirUp, Moving)
	// No requests above – should stop at current floor even without a request here
	if !ShouldStop(e) {
		t.Error("expected stop when no more requests above while going up")
	}
}

func TestShouldStop_NoMoreRequestsBelow(t *testing.T) {
	e := makeElevator(1, DirDown, Moving)
	// No requests below – should stop at current floor
	if !ShouldStop(e) {
		t.Error("expected stop when no more requests below while going down")
	}
}

// ---- ChooseDirection ----

func TestChooseDirection_IdleGoesUpWhenRequestAbove(t *testing.T) {
	e := makeElevator(0, DirStop, Idle)
	e.Requests[3][int(elevio.BT_Cab)] = true
	dir, state := ChooseDirection(e)
	if dir != DirUp || state != Moving {
		t.Errorf("expected DirUp/Moving, got %v/%v", dir, state)
	}
}

func TestChooseDirection_IdleGoesDownWhenRequestBelow(t *testing.T) {
	e := makeElevator(3, DirStop, Idle)
	e.Requests[0][int(elevio.BT_Cab)] = true
	dir, state := ChooseDirection(e)
	if dir != DirDown || state != Moving {
		t.Errorf("expected DirDown/Moving, got %v/%v", dir, state)
	}
}

func TestChooseDirection_OpenDoorWhenRequestHere(t *testing.T) {
	e := makeElevator(2, DirStop, Idle)
	e.Requests[2][int(elevio.BT_Cab)] = true
	dir, state := ChooseDirection(e)
	if state != DoorOpen {
		t.Errorf("expected DoorOpen, got %v (dir=%v)", state, dir)
	}
}

func TestChooseDirection_ContinueUpWhenRequestsAbove(t *testing.T) {
	e := makeElevator(1, DirUp, Moving)
	e.Requests[3][int(elevio.BT_Cab)] = true
	dir, state := ChooseDirection(e)
	if dir != DirUp || state != Moving {
		t.Errorf("expected DirUp/Moving, got %v/%v", dir, state)
	}
}

func TestChooseDirection_ReverseWhenNoMoreAbove(t *testing.T) {
	e := makeElevator(3, DirUp, Moving)
	e.Requests[0][int(elevio.BT_Cab)] = true
	dir, state := ChooseDirection(e)
	if dir != DirDown || state != Moving {
		t.Errorf("expected DirDown/Moving, got %v/%v", dir, state)
	}
}

func TestChooseDirection_IdleWhenNoRequests(t *testing.T) {
	e := makeElevator(2, DirStop, Idle)
	dir, state := ChooseDirection(e)
	if dir != DirStop || state != Idle {
		t.Errorf("expected DirStop/Idle, got %v/%v", dir, state)
	}
}

// ---- ClearAtCurrentFloor ----

func TestClearAtCurrentFloor_CabAlwaysCleared(t *testing.T) {
	for _, dir := range []Dir{DirUp, DirDown, DirStop} {
		e := makeElevator(2, dir, DoorOpen)
		e.Requests[2][int(elevio.BT_Cab)] = true
		e = ClearAtCurrentFloor(e)
		if e.Requests[2][int(elevio.BT_Cab)] {
			t.Errorf("expected cab request cleared when dir=%v", dir)
		}
	}
}

func TestClearAtCurrentFloor_HallUpClearedWhenGoingUp(t *testing.T) {
	e := makeElevator(2, DirUp, DoorOpen)
	e.Requests[2][int(elevio.BT_HallUp)] = true
	e.Requests[2][int(elevio.BT_HallDown)] = true
	e.Requests[3][int(elevio.BT_Cab)] = true // request above so down is preserved
	e = ClearAtCurrentFloor(e)
	if e.Requests[2][int(elevio.BT_HallUp)] {
		t.Error("expected hall-up request cleared when going up")
	}
	if !e.Requests[2][int(elevio.BT_HallDown)] {
		t.Error("expected hall-down preserved when requests still exist above")
	}
}

func TestClearAtCurrentFloor_HallDownClearedWhenGoingDown(t *testing.T) {
	e := makeElevator(2, DirDown, DoorOpen)
	e.Requests[2][int(elevio.BT_HallDown)] = true
	e.Requests[0][int(elevio.BT_Cab)] = true // request below so up is preserved
	e = ClearAtCurrentFloor(e)
	if e.Requests[2][int(elevio.BT_HallDown)] {
		t.Error("expected hall-down request cleared when going down")
	}
}

func TestClearAtCurrentFloor_BothHallClearedWhenStopped(t *testing.T) {
	e := makeElevator(2, DirStop, Idle)
	e.Requests[2][int(elevio.BT_HallUp)] = true
	e.Requests[2][int(elevio.BT_HallDown)] = true
	e = ClearAtCurrentFloor(e)
	if e.Requests[2][int(elevio.BT_HallUp)] {
		t.Error("expected hall-up cleared when stopped")
	}
	if e.Requests[2][int(elevio.BT_HallDown)] {
		t.Error("expected hall-down cleared when stopped")
	}
}
