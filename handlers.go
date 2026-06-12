package main

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type App struct {
	db        *sql.DB
	templates *template.Template
}

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	rows, err := app.db.Query(`
		SELECT t.id, t.name, t.size_gallons, t.tank_type, t.notes, t.created_at,
		       MAX(p.logged_at) as last_logged
		FROM tanks t
		LEFT JOIN parameters p ON t.id = p.tank_id
		GROUP BY t.id
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type TankWithLastLog struct {
		Tank
		LastLogged *time.Time
	}

	var tanks []TankWithLastLog
	for rows.Next() {
		var t TankWithLastLog
		var lastLogged sql.NullTime
		err := rows.Scan(&t.ID, &t.Name, &t.SizeGallons, &t.TankType, &t.Notes, &t.CreatedAt, &lastLogged)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if lastLogged.Valid {
			t.LastLogged = &lastLogged.Time
		}
		tanks = append(tanks, t)
	}

	app.templates.ExecuteTemplate(w, "index.html", tanks)
}

func (app *App) newTankHandler(w http.ResponseWriter, r *http.Request) {
	app.templates.ExecuteTemplate(w, "base.html", nil)
}

func (app *App) createTankHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("name")
	sizeGallons := r.FormValue("size_gallons")
	tankType := r.FormValue("tank_type")
	notes := r.FormValue("notes")

	if name == "" {
		http.Error(w, "Tank name is required", http.StatusBadRequest)
		return
	}

	var size *float64
	if sizeGallons != "" {
		s, err := strconv.ParseFloat(sizeGallons, 64)
		if err == nil {
			size = &s
		}
	}

	_, err := app.db.Exec(
		"INSERT INTO tanks (name, size_gallons, tank_type, notes) VALUES (?, ?, ?, ?)",
		name, size, tankType, notes,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) tankDetailHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/tanks/")
	parts := strings.Split(idStr, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var tank Tank
	err = app.db.QueryRow(
		"SELECT id, name, size_gallons, tank_type, notes, created_at FROM tanks WHERE id = ?",
		id,
	).Scan(&tank.ID, &tank.Name, &tank.SizeGallons, &tank.TankType, &tank.Notes, &tank.CreatedAt)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get recent parameters
	paramRows, err := app.db.Query(`
		SELECT id, ph, ammonia, nitrite, nitrate, temp_f, notes, logged_at
		FROM parameters
		WHERE tank_id = ?
		ORDER BY logged_at DESC
		LIMIT 10
	`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer paramRows.Close()

	var params []Parameter
	for paramRows.Next() {
		var p Parameter
		err := paramRows.Scan(&p.ID, &p.PH, &p.Ammonia, &p.Nitrite, &p.Nitrate, &p.TempF, &p.Notes, &p.LoggedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		params = append(params, p)
	}

	// Get recent observations
	obsRows, err := app.db.Query(`
		SELECT id, note, observed_at
		FROM observations
		WHERE tank_id = ?
		ORDER BY observed_at DESC
		LIMIT 10
	`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer obsRows.Close()

	var observations []Observation
	for obsRows.Next() {
		var o Observation
		err := obsRows.Scan(&o.ID, &o.Note, &o.ObservedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		observations = append(observations, o)
	}

	data := struct {
		Tank         Tank
		Parameters   []Parameter
		Observations []Observation
	}{
		Tank:         tank,
		Parameters:   params,
		Observations: observations,
	}

	app.templates.ExecuteTemplate(w, "tank.html", data)
}

func (app *App) logFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/tanks/")
	idStr = strings.TrimSuffix(idStr, "/log")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var tank Tank
	err = app.db.QueryRow("SELECT id, name FROM tanks WHERE id = ?", id).Scan(&tank.ID, &tank.Name)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app.templates.ExecuteTemplate(w, "log.html", tank)
}

func (app *App) logParametersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/tanks/")
	idStr = strings.TrimSuffix(idStr, "/log")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	parseFloat := func(s string) *float64 {
		if s == "" {
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil
		}
		return &f
	}

	ph := parseFloat(r.FormValue("ph"))
	ammonia := parseFloat(r.FormValue("ammonia"))
	nitrite := parseFloat(r.FormValue("nitrite"))
	nitrate := parseFloat(r.FormValue("nitrate"))
	tempF := parseFloat(r.FormValue("temp_f"))
	notes := r.FormValue("notes")

	_, err = app.db.Exec(`
		INSERT INTO parameters (tank_id, ph, ammonia, nitrite, nitrate, temp_f, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, ph, ammonia, nitrite, nitrate, tempF, notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tanks/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (app *App) createObservationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/tanks/")
	idStr = strings.TrimSuffix(idStr, "/observations")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	note := r.FormValue("note")
	if note == "" {
		http.Error(w, "Observation note is required", http.StatusBadRequest)
		return
	}

	_, err = app.db.Exec(
		"INSERT INTO observations (tank_id, note) VALUES (?, ?)",
		id, note,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tanks/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}
