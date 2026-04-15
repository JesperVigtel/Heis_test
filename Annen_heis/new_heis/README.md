# Distributed Go Elevator System

This repository contains a distributed-first elevator controller in Go with one process per elevator, started as:

```bash
go run ./cmd/elevator --id 1
```

The design keeps one fault-tolerance idea only: a replicated in-memory request table shared over UDP gossip. There is no leader, no consensus, no event log, no local file persistence, and no per-press order ID.

## Architecture

- `cmd/elevator`
  - flag parsing, lifecycle, startup sync window, wiring
- `internal/model`
  - enums, request keys, versions, snapshots, request table
- `internal/driver`
  - driver interface
- `internal/driver/mock`
  - default simulator with a small stdin command console
- `internal/cluster`
  - UDP snapshot gossip, request-table merge helpers, peer liveness tracking
- `internal/assigner`
  - deterministic hall assignment with stable tie-breaking
- `internal/controller`
  - FSM, request activation and clearing, stop logic, lamp updates

The controller is intentionally single-threaded. Networking and the mock driver feed events into the controller loop, which keeps state updates explicit and deterministic.

## Fault Tolerance Model

Each logical request key maps to one `RequestCell`:

- hall request: `(floor, direction)`
- cab request: `(ownerElevatorID, floor)`

Each cell stores:

- `active bool`
- `version { counter, writerID, bootID }`

`bootID` is regenerated on every process start. Local writes use a single Lamport-style counter per process. Incoming snapshots merge request cells independently, so duplicated, delayed, or reordered packets converge by version order.

Important consequence:

- active cab requests are replicated in memory to peers
- a restarted elevator with the same `--id` can relearn those requests during the startup sync window
- if every node that knew a cab request crashes, that request is lost

There is explicitly no disk-backed recovery anywhere in the system.

## Gossip And Liveness

- every node periodically sends a full snapshot over UDP
- each snapshot contains local soft state plus the replicated request table
- peers time out after about 1 second without updates
- timed-out peers are excluded from hall assignment

On startup, a node waits through a short sync window before accepting button input. During that window it merges peer state and seeds its Lamport counter above the highest observed counter.

## Controller Behavior

- FSM states: `idle`, `moving`, `doorOpen`, `stuck`
- door-open interval: exactly 3 seconds per service cycle
- obstruction keeps the door open past the timer
- once obstruction clears after the deadline, the door closes immediately
- hall clearing is directional
- if both hall directions are active at a reversal floor, the controller clears one direction, keeps the door open for another exact 3-second interval, and then clears the opposite direction

Hall assignment is computed independently on each node from live peer soft state using a small deterministic cost function. Ties are broken by the lowest elevator ID.

## Disconnected Policy

While a node has no live peers:

- already-known active requests continue to be served
- new cab calls are still accepted
- new hall calls are ignored

This keeps hall calls as shared distributed state only, with conservative semantics during partitions.

## Mock Driver

The default runtime uses the mock driver. After startup, you can type commands on stdin:

```text
cab <floor>
hall-up <floor>
hall-down <floor>
obstruction on
obstruction off
status
help
```

Example:

```bash
go run ./cmd/elevator --id 1
go run ./cmd/elevator --id 2
```

Then type `hall-up 1` in one process and watch the hall lamp converge in both.

## Assumptions And Limitations

- the default runtime is the mock driver
- gossip sends periodic full-state snapshots
- new hall calls are ignored while disconnected
- cab recovery after restart depends on at least one peer already holding the active cab state
- the isolated-then-crash case is out of scope
- the gossip socket implementation is aimed at Unix-like systems

## Code Quality Principles

- short focused functions
- explicit dependencies
- one-way data flow into a single controller loop
- deterministic merge and assignment rules
- simple conditionals over clever abstraction
- comments only where intent would otherwise be easy to miss
