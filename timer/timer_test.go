package timer

import (
	"testing"
	"time"
)

func TestTimerNotActiveByDefault(t *testing.T) {
	// A fresh timer should not be timed out
	if TimedOut() {
		t.Error("expected timer not timed out before Start")
	}
}

func TestTimerTimedOut(t *testing.T) {
	Start(0.05) // 50 ms
	if TimedOut() {
		t.Error("expected timer not yet timed out immediately after Start")
	}
	time.Sleep(100 * time.Millisecond)
	if !TimedOut() {
		t.Error("expected timer to have timed out after duration elapsed")
	}
}

func TestTimerStopPreventsTimeout(t *testing.T) {
	Start(0.05)
	Stop()
	time.Sleep(100 * time.Millisecond)
	if TimedOut() {
		t.Error("expected stopped timer to not time out")
	}
}

func TestTimerRestart(t *testing.T) {
	Start(0.05)
	time.Sleep(30 * time.Millisecond)
	// Restart before expiry
	Start(0.2)
	time.Sleep(80 * time.Millisecond)
	if TimedOut() {
		t.Error("expected restarted timer not to time out yet")
	}
	time.Sleep(150 * time.Millisecond)
	if !TimedOut() {
		t.Error("expected restarted timer to have timed out")
	}
	Stop()
}
