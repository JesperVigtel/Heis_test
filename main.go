package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/JesperVigtel/Heis_test/elevator"
	"github.com/JesperVigtel/Heis_test/elevio"
	"github.com/JesperVigtel/Heis_test/hallassigner"
	"github.com/JesperVigtel/Heis_test/network/bcast"
	"github.com/JesperVigtel/Heis_test/network/peers"
	"github.com/JesperVigtel/Heis_test/timer"
)

// ---------- Network message types ----------

// ElevatorStateMsg is broadcast by each elevator to share its current state.
type ElevatorStateMsg struct {
	ID          string
	Floor       int
	Dir         elevio.MotorDirection
	Behaviour   elevator.State
	CabRequests [elevio.NumFloors]bool
}

// HallRequestMsg is broadcast by the master to share hall request assignments.
type HallRequestMsg struct {
	SenderID     string
	HallRequests [elevio.NumFloors][2]bool
}

// AckMsg acknowledges receipt of a hall order.
type AckMsg struct {
	ID    string
	Floor int
	Btn   int
}

// ---------- Ports ----------
const (
	portPeers     = 15511
	portStateBcast = 15512
	portHallBcast  = 15513
	portAckBcast   = 15514
)

func main() {
	var id string
	var serverAddr string
	var numFloors int

	flag.StringVar(&id, "id", "", "Unique elevator ID (required)")
	flag.StringVar(&serverAddr, "server", "localhost:15657", "Elevator server address")
	flag.IntVar(&numFloors, "floors", elevio.NumFloors, "Number of floors")
	flag.Parse()

	if id == "" {
		// Default to hostname if no ID given
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "elevator0"
		}
		id = hostname
	}
	fmt.Printf("Starting elevator ID=%s server=%s floors=%d\n", id, serverAddr, numFloors)

	// --- Connect to elevator hardware ---
	elevio.Init(serverAddr, numFloors)

	// --- Channels for hardware events ---
	chButtons      := make(chan elevio.ButtonEvent, 32)
	chFloor        := make(chan int, 8)
	chObstruction  := make(chan bool, 8)
	chStop         := make(chan bool, 8)

	go elevio.PollButtons(chButtons)
	go elevio.PollFloorSensor(chFloor)
	go elevio.PollObstructionSwitch(chObstruction)
	go elevio.PollStopButton(chStop)

	// --- Network channels ---
	chPeerUpdates  := make(chan peers.PeerUpdate, 8)
	chStateTx      := make(chan ElevatorStateMsg, 8)
	chStateRx      := make(chan ElevatorStateMsg, 8)
	chHallTx       := make(chan HallRequestMsg, 8)
	chHallRx       := make(chan HallRequestMsg, 8)
	chAckTx        := make(chan AckMsg, 8)
	chAckRx        := make(chan AckMsg, 8)

	go peers.Transmitter(portPeers, id, nil)
	go peers.Receiver(portPeers, chPeerUpdates)
	go bcast.Transmitter(portStateBcast, chStateTx)
	go bcast.Receiver(portStateBcast, chStateRx)
	go bcast.Transmitter(portHallBcast, chHallTx)
	go bcast.Receiver(portHallBcast, chHallRx)
	go bcast.Transmitter(portAckBcast, chAckTx)
	go bcast.Receiver(portAckBcast, chAckRx)

	// --- Elevator state ---
	e := elevator.UninitializedElevator()

	// Move to defined position if between floors at startup
	if elevio.GetFloor() == -1 {
		elevator.OnInitBetweenFloors(&e)
		// Wait until we hit a floor
		f := <-chFloor
		elevio.SetMotorDirection(elevio.MD_Stop)
		e.Floor = f
		e.Dir = elevator.DirStop
		e.Behaviour = elevator.Idle
		elevio.SetFloorIndicator(f)
	} else {
		f := elevio.GetFloor()
		e.Floor = f
		elevio.SetFloorIndicator(f)
	}
	fmt.Printf("Elevator initialised at floor %d\n", e.Floor)

	// --- Shared world state ---
	// hallRequests: [floor][0=Up,1=Down] → true if pending
	var hallRequests [elevio.NumFloors][2]bool
	// peerStates: map from peer ID to their last broadcast state
	peerStates := make(map[string]hallassigner.ElevatorState)
	// peerList: list of currently alive peer IDs
	var peerList []string

	// masterID: we use sorted-first peer as master for hall assignments.
	masterID := func() string {
		if len(peerList) == 0 {
			return id
		}
		return peerList[0]
	}

	// Broadcast our state periodically and on change.
	stateTicker := time.NewTicker(200 * time.Millisecond)
	broadcastState := func() {
		var cab [elevio.NumFloors]bool
		for f := 0; f < elevio.NumFloors; f++ {
			cab[f] = e.Requests[f][int(elevio.BT_Cab)]
		}
		chStateTx <- ElevatorStateMsg{
			ID:          id,
			Floor:       e.Floor,
			Dir:         e.Dir,
			Behaviour:   e.Behaviour,
			CabRequests: cab,
		}
	}

	// reassign: compute new hall assignments and apply ours.
	reassign := func() {
		assignments := hallassigner.Assign(hallRequests, peerStates)
		if myAssign, ok := assignments[id]; ok {
			for f := 0; f < elevio.NumFloors; f++ {
				// Hall up
				e.Requests[f][int(elevio.BT_HallUp)] = myAssign[f][int(elevio.BT_HallUp)]
				// Hall down
				e.Requests[f][int(elevio.BT_HallDown)] = myAssign[f][int(elevio.BT_HallDown)]
			}
		}
		elevator.SetAllLights(e)
	}

	// Periodically broadcast hall requests so all nodes stay in sync.
	hallTicker := time.NewTicker(500 * time.Millisecond)

	// Door timer check
	doorTicker := time.NewTicker(50 * time.Millisecond)

	for {
		select {
		// ---- Hardware events ----
		case btn := <-chButtons:
			switch btn.Button {
			case elevio.BT_Cab:
				// Cab calls are handled locally only
				var startTimer bool
				e, startTimer = elevator.OnRequestButtonPress(e, btn.Floor, btn.Button)
				if startTimer {
					timer.Start(e.Config.DoorOpenDuration)
				}
				elevator.SetAllLights(e)
				broadcastState()

			case elevio.BT_HallUp, elevio.BT_HallDown:
				// Hall calls: register and broadcast so master can assign
				hallRequests[btn.Floor][int(btn.Button)] = true
				if masterID() == id {
					// We are master: assign immediately
					reassign()
				}
				// Broadcast updated hall requests
				chHallTx <- HallRequestMsg{SenderID: id, HallRequests: hallRequests}
			}

		case floor := <-chFloor:
			var startTimer bool
			e, startTimer = elevator.OnFloorArrival(e, floor)
			if startTimer {
				timer.Start(e.Config.DoorOpenDuration)
				// Clear hall request from global table when we service it
				clearServedHall(e, floor, &hallRequests)
				if masterID() == id {
					reassign()
				}
				chHallTx <- HallRequestMsg{SenderID: id, HallRequests: hallRequests}
			}
			elevator.SetAllLights(e)
			broadcastState()

		case obstructed := <-chObstruction:
			e.Obstructed = obstructed
			if !obstructed && e.Behaviour == elevator.DoorOpen {
				// Door obstruction cleared; restart the door timer
				timer.Start(e.Config.DoorOpenDuration)
			}

		case <-chStop:
			// Stop button: not required; ignore for now.

		// ---- Door timer ----
		case <-doorTicker.C:
			if timer.TimedOut() {
				timer.Stop()
				var startTimer bool
				e, startTimer = elevator.OnDoorTimeout(e)
				if startTimer {
					timer.Start(e.Config.DoorOpenDuration)
				}
				elevator.SetAllLights(e)
				broadcastState()
			}

		// ---- Network events ----
		case pu := <-chPeerUpdates:
			peerList = pu.Peers
			// Remove stale peer states
			for _, lostID := range pu.Lost {
				delete(peerStates, lostID)
			}
			fmt.Printf("Peers: %v  new=%s lost=%v\n", pu.Peers, pu.New, pu.Lost)
			// Re-assign in case master changed or a peer was lost
			if masterID() == id {
				reassign()
			}

		case stateMsg := <-chStateRx:
			if stateMsg.ID == id {
				continue // our own broadcast
			}
			peerStates[stateMsg.ID] = hallassigner.ElevatorState{
				Behaviour:   stateMsg.Behaviour,
				Floor:       stateMsg.Floor,
				Direction:   stateMsg.Dir,
				CabRequests: stateMsg.CabRequests,
			}
			if masterID() == id {
				reassign()
			}

		case hallMsg := <-chHallRx:
			if hallMsg.SenderID == id {
				continue // our own broadcast
			}
			// Merge incoming hall requests (OR only: a peer may have just
			// started and will have an empty table, so we must never clear
			// a request just because the sender doesn't have it).
			changed := false
			for f := 0; f < elevio.NumFloors; f++ {
				for b := 0; b < 2; b++ {
					if hallMsg.HallRequests[f][b] && !hallRequests[f][b] {
						hallRequests[f][b] = true
						changed = true
					}
				}
			}
			if changed && masterID() == id {
				reassign()
			}

		case <-chAckRx:
			// Acknowledgement: not currently used

		// ---- Periodic broadcasts ----
		case <-stateTicker.C:
			broadcastState()

		case <-hallTicker.C:
			if masterID() == id {
				chHallTx <- HallRequestMsg{SenderID: id, HallRequests: hallRequests}
			}
		}
	}
}

// clearServedHall clears hall requests that have been serviced when the
// elevator arrives at a floor.
func clearServedHall(e elevator.Elevator, floor int, hallRequests *[elevio.NumFloors][2]bool) {
	switch e.Dir {
	case elevator.DirUp:
		hallRequests[floor][int(elevio.BT_HallUp)] = false
		if !requestsAboveOrHere(e, floor) {
			hallRequests[floor][int(elevio.BT_HallDown)] = false
		}
	case elevator.DirDown:
		hallRequests[floor][int(elevio.BT_HallDown)] = false
		if !requestsBelowOrHere(e, floor) {
			hallRequests[floor][int(elevio.BT_HallUp)] = false
		}
	case elevator.DirStop:
		hallRequests[floor][int(elevio.BT_HallUp)] = false
		hallRequests[floor][int(elevio.BT_HallDown)] = false
	}
}

func requestsAboveOrHere(e elevator.Elevator, floor int) bool {
	for f := floor; f < elevio.NumFloors; f++ {
		for b := 0; b < elevator.NumButtons; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func requestsBelowOrHere(e elevator.Elevator, floor int) bool {
	for f := 0; f <= floor; f++ {
		for b := 0; b < elevator.NumButtons; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}
