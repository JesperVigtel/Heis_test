package mock

import (
	"fmt"
	"sync"
	"time"

	"new_heis/internal/driver"
	"new_heis/internal/model"
)

type Config struct {
	Floors         int
	StartFloor     int
	TravelDuration time.Duration
}

type Snapshot struct {
	Floor       int
	Motor       model.Direction
	DoorOpen    bool
	ButtonLamps map[string]bool
}

type Driver struct {
	cfg     Config
	buttons chan driver.ButtonEvent
	floors  chan int
	obstr   chan bool
	stop    chan struct{}
	done    chan struct{}

	mu       sync.Mutex
	floor    int
	motor    model.Direction
	doorOpen bool
	lamps    map[string]bool
	lastMove time.Time
}

func New(cfg Config) *Driver {
	if cfg.Floors == 0 {
		cfg.Floors = model.DefaultFloors
	}
	if cfg.TravelDuration == 0 {
		cfg.TravelDuration = 700 * time.Millisecond
	}

	d := &Driver{
		cfg:      cfg,
		buttons:  make(chan driver.ButtonEvent, 64),
		floors:   make(chan int, 64),
		obstr:    make(chan bool, 16),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		floor:    cfg.StartFloor,
		lamps:    map[string]bool{},
		lastMove: time.Now(),
	}
	go d.run()
	return d
}

func (d *Driver) ButtonEvents() <-chan driver.ButtonEvent { return d.buttons }
func (d *Driver) FloorEvents() <-chan int                 { return d.floors }
func (d *Driver) ObstructionEvents() <-chan bool          { return d.obstr }

func (d *Driver) SetMotorDirection(dir model.Direction) {
	d.mu.Lock()
	if d.motor != dir {
		d.lastMove = time.Now()
	}
	d.motor = dir
	d.mu.Unlock()
}

func (d *Driver) SetButtonLamp(button model.Button, floor int, on bool) {
	d.mu.Lock()
	d.lamps[lampKey(button, floor)] = on
	d.mu.Unlock()
}

func (d *Driver) SetFloorIndicator(floor int) {
	d.mu.Lock()
	d.floor = floor
	d.mu.Unlock()
}

func (d *Driver) SetDoorOpenLamp(on bool) {
	d.mu.Lock()
	d.doorOpen = on
	d.mu.Unlock()
}

func (d *Driver) Press(button model.Button, floor int) {
	d.buttons <- driver.ButtonEvent{Floor: floor, Button: button}
}

func (d *Driver) SetObstruction(on bool) {
	d.obstr <- on
}

func (d *Driver) Snapshot() Snapshot {
	d.mu.Lock()
	defer d.mu.Unlock()

	lamps := make(map[string]bool, len(d.lamps))
	for key, on := range d.lamps {
		lamps[key] = on
	}
	return Snapshot{
		Floor:       d.floor,
		Motor:       d.motor,
		DoorOpen:    d.doorOpen,
		ButtonLamps: lamps,
	}
}

func (d *Driver) Close() error {
	close(d.stop)
	<-d.done
	return nil
}

func (d *Driver) run() {
	defer close(d.done)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-d.stop:
			return
		case now := <-ticker.C:
			d.step(now)
		}
	}
}

func (d *Driver) step(now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.motor == model.DirectionStop {
		d.lastMove = now
		return
	}
	if now.Sub(d.lastMove) < d.cfg.TravelDuration {
		return
	}

	next := d.floor + int(d.motor)
	if next < 0 || next >= d.cfg.Floors {
		d.motor = model.DirectionStop
		return
	}

	d.floor = next
	d.lastMove = now
	select {
	case d.floors <- d.floor:
	default:
	}
}

func lampKey(button model.Button, floor int) string {
	return fmt.Sprintf("%s:%d", button.String(), floor)
}
