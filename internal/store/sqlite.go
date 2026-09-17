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
CREATE TABLE IF NOT EXISTS readings (
	room_id TEXT NOT NULL,
	ppm REAL NOT NULL,
	ts TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS forecasts (
	room_id TEXT NOT NULL,
	ppm_forecast REAL NOT NULL,
	horizon_min REAL NOT NULL,
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

func (s *SQLiteStore) SaveReading(ctx context.Context, r Reading) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO readings (room_id, ppm, ts) VALUES (?, ?, ?)`,
		r.RoomID, r.PPM, r.Time.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) SaveForecast(ctx context.Context, f Forecast) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO forecasts (room_id, ppm_forecast, horizon_min, ts) VALUES (?, ?, ?, ?)`,
		f.RoomID, f.PPMForecast, f.HorizonMin, f.Time.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) SaveCommand(ctx context.Context, c Command) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO commands (room_id, level, ts) VALUES (?, ?, ?)`,
		c.RoomID, c.Level, c.Time.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
