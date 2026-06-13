package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

func initDB(dbPath string) (*sql.DB, error) {
	// _pragma query params are honored by modernc.org/sqlite and ensure
	// foreign key constraints are enforced on every connection.
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
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
		tank_id INTEGER NOT NULL REFERENCES tanks(id) ON DELETE CASCADE,
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
		tank_id INTEGER NOT NULL REFERENCES tanks(id) ON DELETE CASCADE,
		note TEXT NOT NULL,
		observed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS livestock (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tank_id INTEGER NOT NULL REFERENCES tanks(id) ON DELETE CASCADE,
		species TEXT NOT NULL,
		quantity INTEGER NOT NULL DEFAULT 1,
		added_at DATE,
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_parameters_tank_logged ON parameters(tank_id, logged_at DESC);
	CREATE INDEX IF NOT EXISTS idx_observations_tank_observed ON observations(tank_id, observed_at DESC);
	CREATE INDEX IF NOT EXISTS idx_livestock_tank ON livestock(tank_id, created_at DESC);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return db, nil
}

// ListTanks returns all tanks with their most recent parameter log timestamp.
func (app *App) ListTanks() ([]TankWithLastLog, error) {
	rows, err := app.db.Query(`
		SELECT t.id, t.name, COALESCE(t.size_gallons, 0), t.tank_type, COALESCE(t.notes, ''), t.created_at,
		       MAX(p.logged_at) AS last_logged
		FROM tanks t
		LEFT JOIN parameters p ON t.id = p.tank_id
		GROUP BY t.id
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tanks []TankWithLastLog
	for rows.Next() {
		var t TankWithLastLog
		// MAX(logged_at) is an aggregate with no declared column type, so the
		// modernc.org/sqlite driver returns it as a string instead of
		// auto-parsing it as a time.Time. Scan into a string and parse here.
		var lastLogged sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &t.SizeGallons, &t.TankType, &t.Notes, &t.CreatedAt, &lastLogged); err != nil {
			return nil, err
		}
		if lastLogged.Valid {
			if ts, ok := parseSQLiteTime(lastLogged.String); ok {
				t.LastLogged = &ts
			}
		}
		tanks = append(tanks, t)
	}
	return tanks, rows.Err()
}

// parseSQLiteTime parses the timestamp formats SQLite emits for
// CURRENT_TIMESTAMP / datetime() values. Returns (zero, false) if none match.
func parseSQLiteTime(s string) (time.Time, bool) {
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// GetTank fetches one tank by id. Returns sql.ErrNoRows when not found.
func (app *App) GetTank(id int64) (Tank, error) {
	var t Tank
	err := app.db.QueryRow(
		`SELECT id, name, COALESCE(size_gallons, 0), tank_type, COALESCE(notes, ''), created_at
		 FROM tanks WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &t.SizeGallons, &t.TankType, &t.Notes, &t.CreatedAt)
	return t, err
}

// CreateTank inserts a new tank and returns its id.
func (app *App) CreateTank(name, tankType, notes string, size *float64) (int64, error) {
	res, err := app.db.Exec(
		`INSERT INTO tanks (name, size_gallons, tank_type, notes) VALUES (?, ?, ?, ?)`,
		name, size, tankType, notes,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// RecentParameters returns up to `limit` most-recent parameter logs for a tank,
// ordered newest-first.
func (app *App) RecentParameters(tankID int64, limit int) ([]Parameter, error) {
	rows, err := app.db.Query(`
		SELECT id, tank_id, ph, ammonia, nitrite, nitrate, temp_f, COALESCE(notes, ''), logged_at
		FROM parameters
		WHERE tank_id = ?
		ORDER BY logged_at DESC
		LIMIT ?
	`, tankID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Parameter
	for rows.Next() {
		var p Parameter
		if err := rows.Scan(&p.ID, &p.TankID, &p.PH, &p.Ammonia, &p.Nitrite, &p.Nitrate, &p.TempF, &p.Notes, &p.LoggedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// RecentObservations returns up to `limit` most-recent observations for a tank.
func (app *App) RecentObservations(tankID int64, limit int) ([]Observation, error) {
	rows, err := app.db.Query(`
		SELECT id, tank_id, note, observed_at
		FROM observations
		WHERE tank_id = ?
		ORDER BY observed_at DESC
		LIMIT ?
	`, tankID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Observation
	for rows.Next() {
		var o Observation
		if err := rows.Scan(&o.ID, &o.TankID, &o.Note, &o.ObservedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// InsertParameters records a new parameter log for a tank.
func (app *App) InsertParameters(tankID int64, ph, ammonia, nitrite, nitrate, tempF *float64, notes string) error {
	_, err := app.db.Exec(`
		INSERT INTO parameters (tank_id, ph, ammonia, nitrite, nitrate, temp_f, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, tankID, ph, ammonia, nitrite, nitrate, tempF, notes)
	return err
}

// InsertObservation records a new observation for a tank.
func (app *App) InsertObservation(tankID int64, note string) error {
	_, err := app.db.Exec(
		`INSERT INTO observations (tank_id, note) VALUES (?, ?)`,
		tankID, note,
	)
	return err
}

// DeleteTank removes a tank and all its child rows in a single transaction.
// With ON DELETE CASCADE + foreign_keys=ON, the child rows go automatically,
// but the explicit transaction keeps the operation atomic across future changes.
func (app *App) DeleteTank(id int64) error {
	tx, err := app.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM tanks WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// ListLivestock returns all stocking entries for a tank, newest first.
func (app *App) ListLivestock(tankID int64) ([]Livestock, error) {
	rows, err := app.db.Query(`
		SELECT id, tank_id, species, quantity, added_at, COALESCE(notes, ''), created_at
		FROM livestock
		WHERE tank_id = ?
		ORDER BY created_at DESC
	`, tankID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Livestock
	for rows.Next() {
		var l Livestock
		// added_at is a plain DATE column; we let the driver scan it into
		// sql.NullString and convert manually so a NULL becomes a nil *time.Time.
		var addedAt sql.NullString
		if err := rows.Scan(&l.ID, &l.TankID, &l.Species, &l.Quantity, &addedAt, &l.Notes, &l.CreatedAt); err != nil {
			return nil, err
		}
		if addedAt.Valid {
			if t, err := time.Parse("2006-01-02", addedAt.String); err == nil {
				l.AddedAt = &t
			} else if t, ok := parseSQLiteTime(addedAt.String); ok {
				l.AddedAt = &t
			}
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// InsertLivestock adds a stocking entry for the given tank.
func (app *App) InsertLivestock(tankID int64, species string, qty int, addedAt *time.Time, notes string) error {
	var added any
	if addedAt != nil {
		added = addedAt.Format("2006-01-02")
	}
	_, err := app.db.Exec(
		`INSERT INTO livestock (tank_id, species, quantity, added_at, notes) VALUES (?, ?, ?, ?, ?)`,
		tankID, species, qty, added, notes,
	)
	return err
}

// DeleteLivestock removes a stocking entry. Returns sql.ErrNoRows if the entry
// does not belong to the supplied tank (defense against id tampering).
func (app *App) DeleteLivestock(tankID, id int64) error {
	res, err := app.db.Exec(`DELETE FROM livestock WHERE id = ? AND tank_id = ?`, id, tankID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
