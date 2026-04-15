package driver

import "new_heis/internal/model"

type ButtonEvent struct {
	Floor  int
	Button model.Button
}

type Device interface {
	ButtonEvents() <-chan ButtonEvent
	FloorEvents() <-chan int
	ObstructionEvents() <-chan bool
	SetMotorDirection(model.Direction)
	SetButtonLamp(model.Button, int, bool)
	SetFloorIndicator(int)
	SetDoorOpenLamp(bool)
}
