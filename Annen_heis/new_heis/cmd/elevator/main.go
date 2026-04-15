package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"new_heis/internal/cluster"
	"new_heis/internal/controller"
	"new_heis/internal/driver/mock"
	"new_heis/internal/model"
)

func main() {
	var (
		id             = flag.Int("id", -1, "elevator id")
		port           = flag.Int("gossip-port", 31111, "UDP gossip port")
		floors         = flag.Int("floors", model.DefaultFloors, "number of floors")
		startFloor     = flag.Int("start-floor", 0, "initial mock floor")
		syncWindow     = flag.Duration("sync-window", 350*time.Millisecond, "startup sync window")
		gossipInterval = flag.Duration("gossip-interval", 150*time.Millisecond, "snapshot broadcast interval")
		travelTime     = flag.Duration("travel-time", 700*time.Millisecond, "mock floor travel duration")
	)
	flag.Parse()

	if *id < 0 {
		log.Fatal("--id is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	now := time.Now()
	bootID := model.NewBootID(now)

	drv := mock.New(mock.Config{
		Floors:         *floors,
		StartFloor:     *startFloor,
		TravelDuration: *travelTime,
	})
	defer drv.Close()

	ctrl := controller.New(controller.Config{
		ElevatorID: *id,
		BootID:     bootID,
		Floors:     *floors,
		StartFloor: *startFloor,
	}, drv, now)

	node, err := cluster.Start(ctx, cluster.Config{
		Port:     *port,
		Interval: *gossipInterval,
	})
	if err != nil {
		log.Fatalf("start gossip: %v", err)
	}
	defer node.Close()
	node.SetLocal(ctrl.Snapshot())

	if err := startupSync(ctx, ctrl, node, *syncWindow); err != nil {
		log.Fatalf("startup sync: %v", err)
	}
	ctrl.SeedClockFromObserved()
	ctrl.EnableButtons()
	ctrl.HandleTick(time.Now())
	node.SetLocal(ctrl.Snapshot())

	mock.StartConsole(ctx, drv, os.Stdin, os.Stdout)
	fmt.Printf("elevator %d started with boot %s on gossip port %d\n", *id, bootID, *port)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case snapshot := <-node.Incoming():
			ctrl.ApplyRemote(time.Now(), snapshot)
		case event := <-drv.ButtonEvents():
			ctrl.HandleButton(time.Now(), event)
		case floor := <-drv.FloorEvents():
			ctrl.HandleFloor(time.Now(), floor)
		case obstructed := <-drv.ObstructionEvents():
			ctrl.HandleObstruction(time.Now(), obstructed)
		case now := <-ticker.C:
			ctrl.HandleTick(now)
		}
		node.SetLocal(ctrl.Snapshot())
	}
}

func startupSync(ctx context.Context, ctrl *controller.Controller, node *cluster.Node, window time.Duration) error {
	deadline := time.NewTimer(window)
	defer deadline.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return nil
		case snapshot := <-node.Incoming():
			ctrl.ApplyRemote(time.Now(), snapshot)
			node.SetLocal(ctrl.Snapshot())
		}
	}
}
