// Package store defines how CO2 readings, occupancy counts, and ventilation
// commands are persisted, each as the message type from internal/message.
// Store is the contract other components depend on; sqlite.go holds the
// current implementation. A different backend later only needs a new file
// satisfying the same contract, with no change to any caller.
package store

import (
	"context"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/message"
)

// Store is the only way any component may persist a reading, occupancy count,
// or command (see project_notes.md §7).
type Store interface {
	SaveCO2Reading(ctx context.Context, r message.CO2Reading) error
	SaveOccupancy(ctx context.Context, o message.OccupancyReading) error
	SaveCommand(ctx context.Context, c message.VentilationCommand) error

	// CO2ReadingsSince returns one room's CO2 readings from since onwards,
	// oldest first. No readings gives an empty list, not nil and not an error.
	CO2ReadingsSince(ctx context.Context, roomID string, since time.Time) ([]message.CO2Reading, error)

	// OccupancySince returns one room's occupancy counts from since onwards,
	// oldest first. No counts gives an empty list, not nil and not an error.
	OccupancySince(ctx context.Context, roomID string, since time.Time) ([]message.OccupancyReading, error)

	// CommandsInForceSince returns one room's commands from since onwards,
	// oldest first, led by the last command before since, as a command holds
	// until the next one. No commands gives an empty list, not nil and not an
	// error.
	CommandsInForceSince(ctx context.Context, roomID string, since time.Time) ([]message.VentilationCommand, error)

	// Latest returns the newest timestamp stored in any table, and false when
	// nothing is stored yet.
	Latest(ctx context.Context) (time.Time, bool, error)

	// DeleteBefore deletes every CO2 reading, occupancy count and command
	// older than before, except each room's last command before it, which is
	// still in force. It returns how many rows it deleted.
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)

	Close() error
}
