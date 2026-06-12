package main

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Tank struct {
	ID          int64
	Name        string
	SizeGallons float64
	TankType    string
	Notes       string
	CreatedAt   time.Time
}

type Parameter struct {
	ID       int64
	TankID   int64
	PH       *float64
	Ammonia  *float64
	Nitrite  *float64
	Nitrate  *float64
	TempF    *float64
	Notes    string
	LoggedAt time.Time
}

type Observation struct {
	ID         int64
	TankID     int64
	Note       string
	ObservedAt time.Time
}

func initDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS tanks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		size_gallons REAL,
		tank_type TEXT DEFAULT 'freshwater',
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS parameters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tank_id INTEGER NOT NULL REFERENCES tanks(id),
		ph REAL,
		ammonia REAL,
		nitrite REAL,
		nitrate REAL,
		temp_f REAL,
		notes TEXT,
		logged_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS observations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tank_id INTEGER NOT NULL REFERENCES tanks(id),
		note TEXT NOT NULL,
		observed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return db, nil
}
