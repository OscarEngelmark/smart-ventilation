// Package roomtime is the simulation's shared clock. start.sh records three
// values for a session, and every process computes room time from them the
// same way, so all processes agree without talking to each other and a
// restarted one rejoins the same clock. Reasoning: project_notes.md §6.
package roomtime

import (
	"log"
	"os"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/env"
)

// Clock turns the wall clock into room time for one session.
type Clock struct {
	sessionStart time.Time // wall-clock moment the session started
	roomStart    time.Time // room time at that moment
	speed        float64   // room seconds per wall-clock second
}

// New returns the clock of a session that started at sessionStart on the
// wall clock, at room time roomStart, running speed times faster than the
// wall clock.
func New(sessionStart, roomStart time.Time, speed float64) *Clock {
	return &Clock{sessionStart: sessionStart, roomStart: roomStart, speed: speed}
}

// FromEnv returns the clock start.sh set up, read from SIM_SESSION_START,
// SIM_ROOM_START (both RFC 3339) and SIM_SPEED. When none is set, as when a
// service runs outside a session, room time is the wall clock. A value that
// can't be read stops the service, as in package env.
func FromEnv() *Clock {
	sessionText := os.Getenv("SIM_SESSION_START")
	roomText := os.Getenv("SIM_ROOM_START")
	if sessionText == "" && roomText == "" && os.Getenv("SIM_SPEED") == "" {
		now := time.Now()
		return New(now, now, 1)
	}

	sessionStart := mustParse("SIM_SESSION_START", sessionText)
	roomStart := mustParse("SIM_ROOM_START", roomText)
	speed := env.Float("SIM_SPEED", 0)
	if speed <= 0 {
		log.Fatalf("SIM_SPEED: must be above 0, got %v", speed)
	}
	return New(sessionStart, roomStart, speed)
}

func mustParse(key, text string) time.Time {
	t, err := time.Parse(time.RFC3339, text)
	if err != nil {
		log.Fatalf("%s: %v", key, err)
	}
	return t
}

// Now is the room time at this moment, in the local time zone so a schedule
// reads the right hour and weekday.
func (c *Clock) Now() time.Time {
	return c.At(time.Now())
}

// At is the room time when the wall clock reads wall.
func (c *Clock) At(wall time.Time) time.Time {
	elapsed := wall.Sub(c.sessionStart)
	roomElapsed := time.Duration(float64(elapsed) * c.speed)
	return c.roomStart.Add(roomElapsed).In(time.Local)
}

// Wall is how long d of room time takes on the wall clock.
func (c *Clock) Wall(d time.Duration) time.Duration {
	return time.Duration(float64(d) / c.speed)
}

// NewTicker returns a ticker that fires every d of room time.
func (c *Clock) NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(c.Wall(d))
}
