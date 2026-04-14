// Package timer provides a simple resettable timer for the door open timeout.
package timer

import (
	"time"
)

var (
	timerEndTime time.Time
	timerActive  bool
)

// Start starts (or restarts) the timer with the given duration in seconds.
func Start(duration float64) {
	timerEndTime = time.Now().Add(time.Duration(duration * float64(time.Second)))
	timerActive = true
}

// Stop stops the timer.
func Stop() {
	timerActive = false
}

// TimedOut returns true if the timer has expired.
func TimedOut() bool {
	return timerActive && time.Now().After(timerEndTime)
}
