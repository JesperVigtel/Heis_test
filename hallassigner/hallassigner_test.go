package hallassigner

import (
	"testing"

	"github.com/JesperVigtel/Heis_test/elevator"
	"github.com/JesperVigtel/Heis_test/elevio"
)

// ---- cost ----

func TestCost_ElevatorAtTargetSameDirection(t *testing.T) {
	e := ElevatorState{
		Behaviour: elevator.Idle,
		Floor:     2,
		Direction: elevio.MD_Up,
	}
	// Already at floor 2, requesting up → cost should be very low
	c := cost(e, 2, elevio.BT_HallUp)
	if c != 0 {
		t.Errorf("expected cost 0, got %d", c)
	}
}

func TestCost_ElevatorAtTargetStopped(t *testing.T) {
	e := ElevatorState{
		Behaviour: elevator.Idle,
		Floor:     1,
		Direction: elevio.MD_Stop,
	}
	c := cost(e, 1, elevio.BT_HallUp)
	if c != 0 {
		t.Errorf("expected cost 0 for stopped elevator at target, got %d", c)
	}
}

func TestCost_ElevatorBelowTargetMovingUp(t *testing.T) {
	e := ElevatorState{
		Behaviour: elevator.Moving,
		Floor:     0,
		Direction: elevio.MD_Up,
	}
	c1 := cost(e, 1, elevio.BT_HallUp)
	c2 := cost(e, 2, elevio.BT_HallUp)
	if c1 >= c2 {
		t.Errorf("expected cost to floor 1 (%d) < cost to floor 2 (%d)", c1, c2)
	}
}

func TestCost_DoorOpenAddsExtraCost(t *testing.T) {
	base := ElevatorState{
		Behaviour: elevator.Moving,
		Floor:     0,
		Direction: elevio.MD_Up,
	}
	withDoor := ElevatorState{
		Behaviour: elevator.DoorOpen,
		Floor:     0,
		Direction: elevio.MD_Up,
	}
	c1 := cost(base, 2, elevio.BT_HallUp)
	c2 := cost(withDoor, 2, elevio.BT_HallUp)
	if c2 <= c1 {
		t.Errorf("expected door-open cost (%d) > base cost (%d)", c2, c1)
	}
}

// ---- Assign ----

func TestAssign_NoPeers_EmptyResult(t *testing.T) {
	var hall [elevio.NumFloors][2]bool
	hall[1][0] = true
	result := Assign(hall, nil)
	if len(result) != 0 {
		t.Errorf("expected empty result with no peers, got %v", result)
	}
}

func TestAssign_SingleElevator_GetsAllHallRequests(t *testing.T) {
	var hall [elevio.NumFloors][2]bool
	hall[0][0] = true // floor 0, up
	hall[2][1] = true // floor 2, down

	peers := map[string]ElevatorState{
		"e1": {Behaviour: elevator.Idle, Floor: 0, Direction: elevio.MD_Stop},
	}
	result := Assign(hall, peers)
	m, ok := result["e1"]
	if !ok {
		t.Fatal("expected assignment for e1")
	}
	if !m[0][int(elevio.BT_HallUp)] {
		t.Error("expected hall-up at floor 0 assigned to e1")
	}
	if !m[2][int(elevio.BT_HallDown)] {
		t.Error("expected hall-down at floor 2 assigned to e1")
	}
}

func TestAssign_TwoElevators_CloserGetsRequest(t *testing.T) {
	var hall [elevio.NumFloors][2]bool
	hall[3][0] = true // floor 3, up

	peers := map[string]ElevatorState{
		"close": {Behaviour: elevator.Idle, Floor: 3, Direction: elevio.MD_Stop},
		"far":   {Behaviour: elevator.Idle, Floor: 0, Direction: elevio.MD_Stop},
	}
	result := Assign(hall, peers)

	closeMat := result["close"]
	farMat := result["far"]
	if !closeMat[3][int(elevio.BT_HallUp)] {
		t.Error("expected closer elevator to get the hall-up request at floor 3")
	}
	if farMat[3][int(elevio.BT_HallUp)] {
		t.Error("expected farther elevator NOT to get the hall-up request at floor 3")
	}
}

func TestAssign_CabRequestsPreserved(t *testing.T) {
	var hall [elevio.NumFloors][2]bool // no hall requests
	var cabs [elevio.NumFloors]bool
	cabs[2] = true

	peers := map[string]ElevatorState{
		"e1": {Behaviour: elevator.Idle, Floor: 0, Direction: elevio.MD_Stop, CabRequests: cabs},
	}
	result := Assign(hall, peers)
	m := result["e1"]
	if !m[2][int(elevio.BT_Cab)] {
		t.Error("expected cab request at floor 2 preserved in assignment")
	}
}

func TestAssign_NoHallRequests_NothingAssigned(t *testing.T) {
	var hall [elevio.NumFloors][2]bool
	peers := map[string]ElevatorState{
		"e1": {Behaviour: elevator.Idle, Floor: 1, Direction: elevio.MD_Stop},
		"e2": {Behaviour: elevator.Idle, Floor: 2, Direction: elevio.MD_Stop},
	}
	result := Assign(hall, peers)
	for id, m := range result {
		for f := 0; f < elevio.NumFloors; f++ {
			if m[f][int(elevio.BT_HallUp)] || m[f][int(elevio.BT_HallDown)] {
				t.Errorf("elevator %s unexpectedly got hall request at floor %d", id, f)
			}
		}
	}
}
