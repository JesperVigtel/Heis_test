# Heis_test Makefile
# Common tasks for building, running, and testing the elevator system.

BINARY      := heis
SIM_BIN     := simulator/SimElevatorServer
SIM_PORT    := 15657
SIM_FLOORS  := 4
ELEVATOR_ID ?= elevator0
SERVER      ?= localhost:$(SIM_PORT)

.PHONY: all build run simulator test clean help

all: build

## build: Compile the elevator controller binary
build:
	go build -o $(BINARY) .

## run: Build and start the elevator controller (simulator must already be running)
run: build
	./$(BINARY) -id $(ELEVATOR_ID) -server $(SERVER) -floors $(SIM_FLOORS)

## simulator: Build the SimElevatorServer from source (requires dmd)
simulator:
	bash scripts/setup-simulator.sh

## sim: Launch the simulator (must have run 'make simulator' first)
sim:
	./$(SIM_BIN) --port $(SIM_PORT) --numfloors $(SIM_FLOORS)

## test: Run the full test suite
test:
	go test -v -timeout 60s ./...

## clean: Remove build artefacts
clean:
	rm -f $(BINARY)

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
