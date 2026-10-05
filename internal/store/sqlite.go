package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/OscarEngelmark/smart-ventilation/internal/message"
)

// SQLiteStore is the current implementation of Store, backed by a local
// SQLite file.
type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLite opens the SQLite file at path, creating it and its tables if
// needed. Reads and writes don't block each other (WAL mode), and a request
// that finds the file locked waits up to 5 s instead of failing at once.
// Each table holds at most one row per room and timestamp.
// Reasoning: project_notes.md §7.
func OpenSQLite(path string) (*SQLiteStore, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)" // applied to every connection
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func createSchema(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS co2_readings (
	room_id TEXT NOT NULL,
	ppm REAL NOT NULL,
	ts TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS occupancy (
	room_id TEXT NOT NULL,
	count INTEGER NOT NULL,
	ts TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS commands (
	room_id TEXT NOT NULL,
	level REAL NOT NULL,
	ts TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS co2_readings_room_ts ON co2_readings (room_id, ts);
CREATE UNIQUE INDEX IF NOT EXISTS occupancy_room_ts ON occupancy (room_id, ts);
CREATE UNIQUE INDEX IF NOT EXISTS commands_room_ts ON commands (room_id, ts);
`
	_, err := db.Exec(schema)
	return err
}

func (s *SQLiteStore) SaveCO2Reading(ctx context.Context, r message.CO2Reading) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO co2_readings (room_id, ppm, ts) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		r.RoomID, r.PPM, r.Ts.UTC().Format(time.RFC3339)) // stored as UTC text, so text order is time order
	return err
}

func (s *SQLiteStore) SaveOccupancy(ctx context.Context, o message.OccupancyReading) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO occupancy (room_id, count, ts) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		o.RoomID, o.Count, o.Ts.UTC().Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) SaveCommand(ctx context.Context, c message.VentilationCommand) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO commands (room_id, level, ts) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		c.RoomID, c.Level, c.Ts.UTC().Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) CO2ReadingsSince(ctx context.Context, roomID string, since time.Time) ([]message.CO2Reading, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT room_id, ppm, ts FROM co2_readings WHERE room_id = ? AND ts >= ? ORDER BY ts`,
		roomID, since.UTC().Format(time.RFC3339)) // timestamps are stored as UTC text, which sorts and compares in time order
	if err != nil {
		return nil, err
	}
	defer rows.Close() // runs when this function returns, however it returns

	readings := []message.CO2Reading{} // empty rather than nil, so a caller's JSON gets [] and not null
	for rows.Next() {
		var r message.CO2Reading
		var ts string
		if err := rows.Scan(&r.RoomID, &r.PPM, &ts); err != nil {
			return nil, err
		}
		r.Ts, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("parse stored timestamp %q: %w", ts, err)
		}
		readings = append(readings, r)
	}
	return readings, rows.Err()
}

func (s *SQLiteStore) OccupancySince(ctx context.Context, roomID string, since time.Time) ([]message.OccupancyReading, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT room_id, count, ts FROM occupancy WHERE room_id = ? AND ts >= ? ORDER BY ts`,
		roomID, since.UTC().Format(time.RFC3339)) // compared as UTC text, as in CO2ReadingsSince
	if err != nil {
		return nil, err
	}
	defer rows.Close() // runs when this function returns, however it returns

	counts := []message.OccupancyReading{} // empty rather than nil, as in CO2ReadingsSince
	for rows.Next() {
		var o message.OccupancyReading
		var ts string
		if err := rows.Scan(&o.RoomID, &o.Count, &ts); err != nil {
			return nil, err
		}
		o.Ts, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("parse stored timestamp %q: %w", ts, err)
		}
		counts = append(counts, o)
	}
	return counts, rows.Err()
}

func (s *SQLiteStore) CommandsInForceSince(ctx context.Context, roomID string, since time.Time) ([]message.VentilationCommand, error) {
	from := since.UTC().Format(time.RFC3339) // compared as UTC text, as in CO2ReadingsSince
	rows, err := s.db.QueryContext(ctx, `
SELECT room_id, level, ts FROM (
	SELECT room_id, level, ts FROM commands
	WHERE room_id = ? AND ts < ? ORDER BY ts DESC LIMIT 1
)
UNION ALL
SELECT room_id, level, ts FROM commands WHERE room_id = ? AND ts >= ?
ORDER BY ts`,
		roomID, from, roomID, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // runs when this function returns, however it returns

	commands := []message.VentilationCommand{} // empty rather than nil, as in CO2ReadingsSince
	for rows.Next() {
		var c message.VentilationCommand
		var ts string
		if err := rows.Scan(&c.RoomID, &c.Level, &ts); err != nil {
			return nil, err
		}
		c.Ts, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("parse stored timestamp %q: %w", ts, err)
		}
		commands = append(commands, c)
	}
	return commands, rows.Err()
}

func (s *SQLiteStore) Latest(ctx context.Context) (time.Time, bool, error) {
	var ts sql.NullString // NULL when every table is empty
	err := s.db.QueryRowContext(ctx, `SELECT MAX(ts) FROM (
		SELECT ts FROM co2_readings UNION ALL
		SELECT ts FROM occupancy UNION ALL
		SELECT ts FROM commands
	)`).Scan(&ts)
	if err != nil {
		return time.Time{}, false, err
	}
	if !ts.Valid {
		return time.Time{}, false, nil
	}
	latest, err := time.Parse(time.RFC3339, ts.String)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse stored timestamp %q: %w", ts.String, err)
	}
	return latest, true, nil
}

func (s *SQLiteStore) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	cutoff := before.UTC().Format(time.RFC3339) // compared as UTC text, as in CO2ReadingsSince
	queries := []string{
		`DELETE FROM co2_readings WHERE ts < ?`,
		`DELETE FROM occupancy WHERE ts < ?`,
		// Older than the room's last command before the cutoff, which is kept.
		`DELETE FROM commands WHERE ts < (
	SELECT MAX(ts) FROM commands AS kept
	WHERE kept.room_id = commands.room_id AND kept.ts < ?
)`,
	}

	var deleted int64
	for _, query := range queries {
		result, err := s.db.ExecContext(ctx, query, cutoff)
		if err != nil {
			return deleted, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return deleted, err
		}
		deleted += n
	}
	return deleted, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
