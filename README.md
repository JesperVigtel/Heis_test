# Heis_test

A multi-elevator control system written in Go, built to run against the
[TTK4145 Simulator-v2](https://github.com/TTK4145/Simulator-v2).

---

## Quick-start (macOS)

### 1 – Prerequisites

| Tool | Install |
|------|---------|
| Go ≥ 1.21 | `brew install go` |
| dmd (D compiler) | `brew install dmd` |
| git | bundled with Xcode CLT |

> **Homebrew not installed?** Get it at <https://brew.sh>

### 2 – Clone this repo

```bash
git clone https://github.com/JesperVigtel/Heis_test.git
cd Heis_test
```

### 3 – Build the simulator

The [TTK4145 Simulator-v2](https://github.com/TTK4145/Simulator-v2) has no
pre-built macOS binary, so we compile it from source. The helper script does
everything automatically:

```bash
bash scripts/setup-simulator.sh
```

This clones the simulator source and produces `simulator/SimElevatorServer`.

### 4 – Run (two terminal windows)

**Terminal 1 – simulator:**
```bash
make sim
```

The simulator window shows an ASCII display of the elevator and accepts
keyboard input to press buttons:

| Action | Key |
|--------|-----|
| Hall Up (floors 0-2) | `q w e r` |
| Hall Down (floors 1-3) | `s d f g` |
| Cab call (floors 0-3) | `z x c v` |
| Stop button | `p` |
| Obstruction | `-` |
| Motor override up/stop/down | `9 8 7` |

**Terminal 2 – elevator controller:**
```bash
make run
```

Or with explicit options:
```bash
./heis -id elevator0 -server localhost:15657 -floors 4
```

### 5 – Multiple elevators on one machine

Start one simulator per elevator (each on a different port), then start one
controller per simulator:

```bash
# Terminal 1
./simulator/SimElevatorServer --port 15657 --numfloors 4

# Terminal 2
./simulator/SimElevatorServer --port 15658 --numfloors 4

# Terminal 3
go run . -id elev0 -server localhost:15657

# Terminal 4
go run . -id elev1 -server localhost:15658
```

The controllers discover each other automatically over UDP broadcast and share
hall-call assignments.

---

## Build & test

```bash
make build          # compile → ./heis
make test           # run all unit + integration tests
make clean          # remove ./heis
```

---

## Command-line flags

| Flag | Default | Description |
|------|---------|-------------|
| `-id` | hostname | Unique identifier for this elevator |
| `-server` | `localhost:15657` | Address of the simulator / hardware server |
| `-floors` | `4` | Number of floors (must match simulator) |

---

## Project layout

```
main.go            – entry point, network + hardware event loop
elevator/          – single-elevator FSM (types, requests, door timer)
elevio/            – TCP driver (implements the 4-byte simulator protocol)
hallassigner/      – cost-based hall-call assignment for multiple elevators
network/bcast/     – UDP broadcast for peer-to-peer state sharing
network/peers/     – peer presence tracking
timer/             – resettable door-open timer
scripts/           – helper scripts
Makefile           – convenient build/run/test targets
```