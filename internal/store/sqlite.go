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

func OpenSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
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
`
	_, err := db.Exec(schema)
	return err
}

func (s *SQLiteStore) SaveCO2Reading(ctx context.Context, r CO2Reading) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO co2_readings (room_id, ppm, ts) VALUES (?, ?, ?)`,
		r.RoomID, r.PPM, r.Time.UTC().Format(time.RFC3339)) // stored as UTC text, so text order is time order
	return err
}

func (s *SQLiteStore) SaveOccupancy(ctx context.Context, o Occupancy) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO occupancy (room_id, count, ts) VALUES (?, ?, ?)`,
		o.RoomID, o.Count, o.Time.UTC().Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) SaveCommand(ctx context.Context, c Command) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO commands (room_id, level, ts) VALUES (?, ?, ?)`,
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
