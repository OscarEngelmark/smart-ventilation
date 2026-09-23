// Package store defines how sensor readings, forecasts, and ventilation
// commands are persisted. Store is the contract other components depend
// on; sqlite.go holds the current implementation. A different backend
// later only needs a new file satisfying the same contract, with no
// change to any caller.
package store

import (
	"context"
	"time"
)

type Reading struct {
	RoomID string
	PPM    float64
	Time   time.Time
}

type Forecast struct {
	RoomID      string
	PPMForecast float64
	HorizonMin  float64
	Time        time.Time
}

type Command struct {
	RoomID string
	Level  float64
	Time   time.Time
}

// Store is the only way any component may persist a reading, forecast, or
// command (see project_notes.md §7).
type Store interface {
	SaveReading(ctx context.Context, r Reading) error
	SaveForecast(ctx context.Context, f Forecast) error
	SaveCommand(ctx context.Context, c Command) error

	// ReadingsSince returns one room's readings from since onwards, oldest
	// first. An empty result is not an error.
	ReadingsSince(ctx context.Context, roomID string, since time.Time) ([]Reading, error)

	Close() error
}
