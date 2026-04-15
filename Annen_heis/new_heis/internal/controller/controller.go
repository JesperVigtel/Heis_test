package controller

import (
	"time"

	"new_heis/internal/assigner"
	"new_heis/internal/cluster"
	"new_heis/internal/driver"
	"new_heis/internal/model"
)

type Config struct {
	ElevatorID       int
	BootID           string
	Floors           int
	StartFloor       int
	DoorOpenDuration time.Duration
	PeerTimeout      time.Duration
	StuckTimeout     time.Duration
}

type Controller struct {
	cfg   Config
	dev   driver.Device
	clock model.Clock
	reqs  model.RequestTable
	peers *cluster.PeerTracker
	car   model.SoftState

	ready        bool
	doorDeadline time.Time
	lastMove     time.Time
	travelDir    model.Direction
	doorDir      model.Direction
}

func New(cfg Config, dev driver.Device, now time.Time) *Controller {
	if cfg.Floors == 0 {
		cfg.Floors = model.DefaultFloors
	}
	if cfg.DoorOpenDuration == 0 {
		cfg.DoorOpenDuration = 3 * time.Second
	}
	if cfg.PeerTimeout == 0 {
		cfg.PeerTimeout = time.Second
	}
	if cfg.StuckTimeout == 0 {
		cfg.StuckTimeout = 5 * time.Second
	}

	c := &Controller{
		cfg:   cfg,
		dev:   dev,
		clock: model.NewClock(cfg.ElevatorID, cfg.BootID),
		reqs:  model.NewRequestTable(),
		peers: cluster.NewPeerTracker(cfg.PeerTimeout),
		car: model.SoftState{
			ID:        cfg.ElevatorID,
			BootID:    cfg.BootID,
			Floor:     cfg.StartFloor,
			Direction: model.DirectionStop,
			Behavior:  model.BehaviorIdle,
		},
		lastMove: now,
	}
	c.sync()
	return c
}

func (c *Controller) EnableButtons() {
	c.ready = true
}

func (c *Controller) SeedClockFromObserved() {
	c.clock.Seed(c.reqs.MaxCounter())
}

func (c *Controller) Snapshot() model.Snapshot {
	return model.Snapshot{Soft: c.car, Requests: c.reqs.Entries()}
}

func (c *Controller) ApplyRemote(now time.Time, snap model.Snapshot) {
	changed := cluster.MergeRequests(c.reqs, &c.clock, snap.Requests)
	if snap.Soft.ID != c.cfg.ElevatorID && c.peers.Observe(snap.Soft, now) {
		changed = true
	}
	if changed {
		c.plan(now)
		return
	}
	c.sync()
}

func (c *Controller) HandleButton(now time.Time, event driver.ButtonEvent) {
	if !c.ready || !c.valid(event.Button, event.Floor) {
		return
	}

	switch event.Button {
	case model.ButtonCab:
		c.set(model.CabKey(c.cfg.ElevatorID, event.Floor), true)
	case model.ButtonHallUp, model.ButtonHallDown:
		if !c.peers.Connected() {
			return
		}
		c.set(model.HallKey(event.Floor, event.Button.Direction()), true)
	}
	c.plan(now)
}

func (c *Controller) HandleFloor(now time.Time, floor int) {
	if floor < 0 || floor >= c.cfg.Floors {
		return
	}
	c.car.Floor = floor
	c.lastMove = now
	c.dev.SetFloorIndicator(floor)

	if c.car.Behavior == model.BehaviorMoving || c.car.Behavior == model.BehaviorStuck {
		if c.shouldStop(floor, c.car.Direction, c.assignments()) {
			c.open(now, c.car.Direction)
			return
		}
		if c.car.Behavior == model.BehaviorStuck {
			c.car.Behavior = model.BehaviorMoving
		}
	}
	c.sync()
}

func (c *Controller) HandleObstruction(now time.Time, on bool) {
	c.car.Obstruction = on
	if !on && c.doorStep(now) {
		return
	}
	c.sync()
}

func (c *Controller) HandleTick(now time.Time) {
	expired := c.peers.Expire(now)
	if expired && c.car.Behavior != model.BehaviorDoorOpen {
		c.plan(now)
		return
	}

	if c.car.Behavior == model.BehaviorMoving && now.Sub(c.lastMove) > c.cfg.StuckTimeout {
		c.car.Behavior = model.BehaviorStuck
		c.car.Direction = model.DirectionStop
		c.dev.SetMotorDirection(model.DirectionStop)
		c.sync()
		return
	}

	if c.doorStep(now) {
		return
	}
	if c.car.Behavior == model.BehaviorIdle {
		c.plan(now)
		return
	}
	c.sync()
}

func (c *Controller) doorStep(now time.Time) bool {
	if c.car.Behavior != model.BehaviorDoorOpen || c.car.Obstruction || c.doorDeadline.IsZero() || now.Before(c.doorDeadline) {
		return false
	}

	if dir, ok := c.nextDoorDir(c.assignments()); ok {
		c.doorDir = dir
		c.clearFloor(dir)
		c.doorDeadline = now.Add(c.cfg.DoorOpenDuration)
		c.sync()
		return true
	}

	c.car.DoorOpen = false
	c.car.Behavior = model.BehaviorIdle
	c.doorDir = model.DirectionStop
	c.doorDeadline = time.Time{}
	c.dev.SetDoorOpenLamp(false)
	c.plan(now)
	return true
}

func (c *Controller) plan(now time.Time) {
	if c.car.Behavior == model.BehaviorDoorOpen || c.car.Behavior == model.BehaviorStuck {
		c.sync()
		return
	}

	assignments := c.assignments()
	if c.canServe(c.car.Floor, assignments) {
		c.open(now, c.travelDir)
		return
	}

	dir := c.nextDir(assignments)
	if dir == model.DirectionStop {
		c.car.Behavior = model.BehaviorIdle
		c.car.Direction = model.DirectionStop
		c.dev.SetMotorDirection(model.DirectionStop)
		c.sync()
		return
	}

	c.car.Behavior = model.BehaviorMoving
	c.car.Direction = dir
	c.travelDir = dir
	c.lastMove = now
	c.dev.SetMotorDirection(dir)
	c.sync()
}

func (c *Controller) open(now time.Time, arrival model.Direction) {
	c.car.Behavior = model.BehaviorDoorOpen
	c.car.Direction = model.DirectionStop
	c.car.DoorOpen = true
	c.doorDir = c.pickDoorDir(c.assignments(), arrival)
	c.doorDeadline = now.Add(c.cfg.DoorOpenDuration)

	c.dev.SetMotorDirection(model.DirectionStop)
	c.dev.SetDoorOpenLamp(true)
	c.clearFloor(c.doorDir)
	c.sync()
}

func (c *Controller) nextDoorDir(assignments map[model.RequestKey]int) (model.Direction, bool) {
	floor := c.car.Floor
	switch c.doorDir {
	case model.DirectionUp:
		if c.hall(floor, model.DirectionDown, assignments) && !c.anyAbove(floor, assignments) {
			return model.DirectionDown, true
		}
	case model.DirectionDown:
		if c.hall(floor, model.DirectionUp, assignments) && !c.anyBelow(floor, assignments) {
			return model.DirectionUp, true
		}
	}
	return model.DirectionStop, false
}

func (c *Controller) pickDoorDir(assignments map[model.RequestKey]int, arrival model.Direction) model.Direction {
	floor := c.car.Floor
	up := c.hall(floor, model.DirectionUp, assignments)
	down := c.hall(floor, model.DirectionDown, assignments)

	switch arrival {
	case model.DirectionUp:
		if up {
			return model.DirectionUp
		}
		if down && !c.anyAbove(floor, assignments) {
			return model.DirectionDown
		}
	case model.DirectionDown:
		if down {
			return model.DirectionDown
		}
		if up && !c.anyBelow(floor, assignments) {
			return model.DirectionUp
		}
	default:
		if up && !down {
			return model.DirectionUp
		}
		if down && !up {
			return model.DirectionDown
		}
		if up && down {
			if c.anyAbove(floor, assignments) {
				return model.DirectionUp
			}
			if c.anyBelow(floor, assignments) {
				return model.DirectionDown
			}
			return model.DirectionUp
		}
	}
	return model.DirectionStop
}

func (c *Controller) shouldStop(floor int, dir model.Direction, assignments map[model.RequestKey]int) bool {
	if c.cab(floor) {
		return true
	}
	switch dir {
	case model.DirectionUp:
		return c.hall(floor, model.DirectionUp, assignments) || !c.anyAbove(floor, assignments) && c.hall(floor, model.DirectionDown, assignments)
	case model.DirectionDown:
		return c.hall(floor, model.DirectionDown, assignments) || !c.anyBelow(floor, assignments) && c.hall(floor, model.DirectionUp, assignments)
	default:
		return c.canServe(floor, assignments)
	}
}

func (c *Controller) nextDir(assignments map[model.RequestKey]int) model.Direction {
	switch c.travelDir {
	case model.DirectionUp:
		if c.anyAbove(c.car.Floor, assignments) {
			return model.DirectionUp
		}
		if c.anyBelow(c.car.Floor, assignments) {
			return model.DirectionDown
		}
	case model.DirectionDown:
		if c.anyBelow(c.car.Floor, assignments) {
			return model.DirectionDown
		}
		if c.anyAbove(c.car.Floor, assignments) {
			return model.DirectionUp
		}
	}

	bestFloor, bestDist := -1, c.cfg.Floors+1
	for floor := 0; floor < c.cfg.Floors; floor++ {
		if floor == c.car.Floor || !c.canServe(floor, assignments) {
			continue
		}
		if dist := abs(c.car.Floor - floor); dist < bestDist || dist == bestDist && (bestFloor < 0 || floor < bestFloor) {
			bestFloor, bestDist = floor, dist
		}
	}
	if bestFloor < 0 {
		return model.DirectionStop
	}

	switch {
	case bestFloor > c.car.Floor:
		return model.DirectionUp
	case bestFloor < c.car.Floor:
		return model.DirectionDown
	default:
		return model.DirectionStop
	}
}

func (c *Controller) canServe(floor int, assignments map[model.RequestKey]int) bool {
	return c.cab(floor) || c.hall(floor, model.DirectionUp, assignments) || c.hall(floor, model.DirectionDown, assignments)
}

func (c *Controller) anyAbove(floor int, assignments map[model.RequestKey]int) bool {
	return c.anyFrom(floor, 1, assignments)
}

func (c *Controller) anyBelow(floor int, assignments map[model.RequestKey]int) bool {
	return c.anyFrom(floor, -1, assignments)
}

func (c *Controller) anyFrom(floor, step int, assignments map[model.RequestKey]int) bool {
	for floor += step; floor >= 0 && floor < c.cfg.Floors; floor += step {
		if c.canServe(floor, assignments) {
			return true
		}
	}
	return false
}

func (c *Controller) cab(floor int) bool {
	return c.reqs.Active(model.CabKey(c.cfg.ElevatorID, floor))
}

func (c *Controller) hall(floor int, dir model.Direction, assignments map[model.RequestKey]int) bool {
	key := model.HallKey(floor, dir)
	return c.reqs.Active(key) && assignments[key] == c.cfg.ElevatorID
}

func (c *Controller) assignments() map[model.RequestKey]int {
	return assigner.Assign(c.reqs, c.peers.LiveStates(c.car))
}

func (c *Controller) clearFloor(dir model.Direction) {
	c.set(model.CabKey(c.cfg.ElevatorID, c.car.Floor), false)
	if dir != model.DirectionStop {
		c.set(model.HallKey(c.car.Floor, dir), false)
	}
}

func (c *Controller) set(key model.RequestKey, active bool) {
	if c.reqs.Active(key) == active {
		return
	}
	c.reqs.Set(key, model.RequestCell{Active: active, Version: c.clock.Next()})
}

func (c *Controller) sync() {
	c.dev.SetFloorIndicator(c.car.Floor)
	c.dev.SetDoorOpenLamp(c.car.DoorOpen)
	for floor := 0; floor < c.cfg.Floors; floor++ {
		if floor < c.cfg.Floors-1 {
			c.dev.SetButtonLamp(model.ButtonHallUp, floor, c.reqs.Active(model.HallKey(floor, model.DirectionUp)))
		}
		if floor > 0 {
			c.dev.SetButtonLamp(model.ButtonHallDown, floor, c.reqs.Active(model.HallKey(floor, model.DirectionDown)))
		}
		c.dev.SetButtonLamp(model.ButtonCab, floor, c.reqs.Active(model.CabKey(c.cfg.ElevatorID, floor)))
	}
}

func (c *Controller) valid(button model.Button, floor int) bool {
	if floor < 0 || floor >= c.cfg.Floors {
		return false
	}
	if button == model.ButtonHallUp {
		return floor < c.cfg.Floors-1
	}
	if button == model.ButtonHallDown {
		return floor > 0
	}
	return true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
