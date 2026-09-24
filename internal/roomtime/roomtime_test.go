package roomtime

import (
	"testing"
	"time"
)

// The wall-clock moment the tests' sessions start at.
var sessionStart = time.Date(2026, 9, 25, 7, 0, 0, 0, time.UTC)

// The room time the tests' sessions start at: Monday 5 October, 06:00 UTC.
var roomStart = time.Date(2026, 10, 5, 6, 0, 0, 0, time.UTC)

func TestAtSessionStartRoomTimeIsRoomStart(t *testing.T) {
	c := New(sessionStart, roomStart, 10)

	if got := c.At(sessionStart); !got.Equal(roomStart) {
		t.Errorf("got %v, want %v", got, roomStart)
	}
}

func TestRoomTimeRunsSpeedTimesFaster(t *testing.T) {
	c := New(sessionStart, roomStart, 10)

	got := c.At(sessionStart.Add(6 * time.Minute))
	want := roomStart.Add(60 * time.Minute)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAtSpeedOneRoomTimeKeepsItsDistanceToTheWallClock(t *testing.T) {
	c := New(sessionStart, roomStart, 1)

	wall := sessionStart.Add(3 * time.Hour)
	if got := c.At(wall).Sub(wall); got != roomStart.Sub(sessionStart) {
		t.Errorf("distance %v, want %v", got, roomStart.Sub(sessionStart))
	}
}

func TestWallIsRoomDurationDividedBySpeed(t *testing.T) {
	c := New(sessionStart, roomStart, 10)

	if got := c.Wall(10 * time.Second); got != time.Second {
		t.Errorf("got %v, want 1s", got)
	}
}

func TestWithoutSettingsRoomTimeIsTheWallClock(t *testing.T) {
	t.Setenv("SIM_SESSION_START", "")
	t.Setenv("SIM_ROOM_START", "")
	t.Setenv("SIM_SPEED", "")
	c := FromEnv()

	before := time.Now()
	got := c.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Errorf("room time %v is outside the wall-clock window %v to %v", got, before, after)
	}
}

func TestFromEnvReadsTheSession(t *testing.T) {
	t.Setenv("SIM_SESSION_START", "2026-09-25T07:00:00Z")
	t.Setenv("SIM_ROOM_START", "2026-10-05T08:00:00+02:00")
	t.Setenv("SIM_SPEED", "10")
	c := FromEnv()

	got := c.At(sessionStart.Add(time.Minute))
	want := roomStart.Add(10 * time.Minute)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
