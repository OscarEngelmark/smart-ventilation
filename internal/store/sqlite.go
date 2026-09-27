package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore is the current implementation of Store, backed by a local
// SQLite file.
type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLite opens the SQLite file at path, creating it and its tables if
// needed. Reads and writes don't block each other (WAL mode), and a request
// that finds the file locked waits up to 5 s instead of failing at once.
// Each table holds at most one row per room and timestamp; duplicates
// already in the file are removed on opening, keeping the earliest saved.
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

DELETE FROM co2_readings WHERE rowid NOT IN (
	SELECT MIN(rowid) FROM co2_readings GROUP BY room_id, ts
);
DELETE FROM occupancy WHERE rowid NOT IN (
	SELECT MIN(rowid) FROM occupancy GROUP BY room_id, ts
);
DELETE FROM commands WHERE rowid NOT IN (
	SELECT MIN(rowid) FROM commands GROUP BY room_id, ts
);

CREATE UNIQUE INDEX IF NOT EXISTS co2_readings_room_ts ON co2_readings (room_id, ts);
CREATE UNIQUE INDEX IF NOT EXISTS occupancy_room_ts ON occupancy (room_id, ts);
CREATE UNIQUE INDEX IF NOT EXISTS commands_room_ts ON commands (room_id, ts);
`
	_, err := db.Exec(schema)
	return err
}

func (s *SQLiteStore) SaveCO2Reading(ctx context.Context, r CO2Reading) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO co2_readings (room_id, ppm, ts) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		r.RoomID, r.PPM, r.Time.UTC().Format(time.RFC3339)) // stored as UTC text, so text order is time order
	return err
}

func (s *SQLiteStore) SaveOccupancy(ctx context.Context, o Occupancy) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO occupancy (room_id, count, ts) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		o.RoomID, o.Count, o.Time.UTC().Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) SaveCommand(ctx context.Context, c Command) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO commands (room_id, level, ts) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		c.RoomID, c.Level, c.Time.UTC().Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) CO2ReadingsSince(ctx context.Context, roomID string, since time.Time) ([]CO2Reading, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT room_id, ppm, ts FROM co2_readings WHERE room_id = ? AND ts >= ? ORDER BY ts`,
		roomID, since.UTC().Format(time.RFC3339)) // timestamps are stored as UTC text, which sorts and compares in time order
	if err != nil {
		return nil, err
	}
	defer rows.Close() // runs when this function returns, however it returns

	var readings []CO2Reading
	for rows.Next() {
		var r CO2Reading
		var ts string
		if err := rows.Scan(&r.RoomID, &r.PPM, &ts); err != nil {
			return nil, err
		}
		r.Time, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("parse stored timestamp %q: %w", ts, err)
		}
		readings = append(readings, r)
	}
	return readings, rows.Err()
}

func (s *SQLiteStore) OccupancySince(ctx context.Context, roomID string, since time.Time) ([]Occupancy, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT room_id, count, ts FROM occupancy WHERE room_id = ? AND ts >= ? ORDER BY ts`,
		roomID, since.UTC().Format(time.RFC3339)) // compared as UTC text, as in CO2ReadingsSince
	if err != nil {
		return nil, err
	}
	defer rows.Close() // runs when this function returns, however it returns

	var counts []Occupancy
	for rows.Next() {
		var o Occupancy
		var ts string
		if err := rows.Scan(&o.RoomID, &o.Count, &ts); err != nil {
			return nil, err
		}
		o.Time, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("parse stored timestamp %q: %w", ts, err)
		}
		counts = append(counts, o)
	}
	return counts, rows.Err()
}

func (s *SQLiteStore) CommandsInForceSince(ctx context.Context, roomID string, since time.Time) ([]Command, error) {
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

	var commands []Command
	for rows.Next() {
		var c Command
		var ts string
		if err := rows.Scan(&c.RoomID, &c.Level, &ts); err != nil {
			return nil, err
		}
		c.Time, err = time.Parse(time.RFC3339, ts)
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

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
