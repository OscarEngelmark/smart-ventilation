// Package store defines how CO2 readings, occupancy counts, forecasts, and
// ventilation commands are persisted. Store is the contract other components depend
// on; sqlite.go holds the current implementation. A different backend
// later only needs a new file satisfying the same contract, with no
// change to any caller.
package store

import (
	"context"
	"time"
)

type CO2Reading struct {
	RoomID string
	PPM    float64
	Time   time.Time
}

type Occupancy struct {
	RoomID string
	Count  int
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

// Store is the only way any component may persist a reading, occupancy count,
// forecast, or command (see project_notes.md §7).
type Store interface {
	SaveCO2Reading(ctx context.Context, r CO2Reading) error
	SaveOccupancy(ctx context.Context, o Occupancy) error
	SaveForecast(ctx context.Context, f Forecast) error
	SaveCommand(ctx context.Context, c Command) error

	// CO2ReadingsSince returns one room's CO2 readings from since onwards,
	// oldest first. An empty result is not an error.
	CO2ReadingsSince(ctx context.Context, roomID string, since time.Time) ([]CO2Reading, error)

	// OccupancySince returns one room's occupancy counts from since onwards,
	// oldest first. An empty result is not an error.
	OccupancySince(ctx context.Context, roomID string, since time.Time) ([]Occupancy, error)

	Close() error
}
