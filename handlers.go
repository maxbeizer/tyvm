package main

import (
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

// render executes the named template, logging — and reporting — failures
// instead of leaving a half-written response on the floor.
func (app *App) render(w http.ResponseWriter, name string, data any) {
	if err := app.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
		// Headers may already be flushed; this is best-effort.
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// pathID extracts the {id} path value as int64.
func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// parseOptionalFloat parses a possibly-empty form value. An empty string
// returns (nil, nil); a non-empty but invalid string returns an error.
func parseOptionalFloat(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) {
	tanks, err := app.ListTanks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	app.render(w, "index.html", struct {
		Tanks     []TankWithLastLog
		CSRFToken string
	}{Tanks: tanks, CSRFToken: csrfToken(r)})
}

func (app *App) newTankHandler(w http.ResponseWriter, r *http.Request) {
	app.render(w, "new_tank.html", struct{ CSRFToken string }{CSRFToken: csrfToken(r)})
}

func (app *App) createTankHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Tank name is required", http.StatusBadRequest)
		return
	}

	size, err := parseOptionalFloat(r.FormValue("size_gallons"))
	if err != nil {
		http.Error(w, "Invalid size_gallons", http.StatusBadRequest)
		return
	}

	if _, err := app.CreateTank(name, r.FormValue("tank_type"), r.FormValue("notes"), size); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) tankDetailHandler(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	tank, err := app.GetTank(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	params, err := app.RecentParameters(id, 30)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	observations, err := app.RecentObservations(id, 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract non-nil values for each parameter type in chronological order.
	var phVals, ammoniaVals, nitriteVals, nitrateVals, tempVals []float64
	for i := len(params) - 1; i >= 0; i-- {
		p := params[i]
		if p.PH != nil {
			phVals = append(phVals, *p.PH)
		}
		if p.Ammonia != nil {
			ammoniaVals = append(ammoniaVals, *p.Ammonia)
		}
		if p.Nitrite != nil {
			nitriteVals = append(nitriteVals, *p.Nitrite)
		}
		if p.Nitrate != nil {
			nitrateVals = append(nitrateVals, *p.Nitrate)
		}
		if p.TempF != nil {
			tempVals = append(tempVals, *p.TempF)
		}
	}

	data := struct {
		Tank         Tank
		Parameters   []Parameter
		Observations []Observation
		Sparklines   map[string]template.HTML
		CSRFToken    string
	}{
		Tank:         tank,
		Parameters:   params,
		Observations: observations,
		Sparklines: map[string]template.HTML{
			"ph":      template.HTML(sparkline(phVals, 80, 24)),
			"ammonia": template.HTML(sparkline(ammoniaVals, 80, 24)),
			"nitrite": template.HTML(sparkline(nitriteVals, 80, 24)),
			"nitrate": template.HTML(sparkline(nitrateVals, 80, 24)),
			"temp":    template.HTML(sparkline(tempVals, 80, 24)),
		},
		CSRFToken: csrfToken(r),
	}

	app.render(w, "tank.html", data)
}

func (app *App) logFormHandler(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	tank, err := app.GetTank(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app.render(w, "log.html", struct {
		Tank      Tank
		CSRFToken string
	}{Tank: tank, CSRFToken: csrfToken(r)})
}

func (app *App) logParametersHandler(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	fields := []struct {
		name string
		ptr  **float64
	}{
		{"ph", new(*float64)},
		{"ammonia", new(*float64)},
		{"nitrite", new(*float64)},
		{"nitrate", new(*float64)},
		{"temp_f", new(*float64)},
	}
	for _, f := range fields {
		v, err := parseOptionalFloat(r.FormValue(f.name))
		if err != nil {
			http.Error(w, "Invalid value for "+f.name, http.StatusBadRequest)
			return
		}
		*f.ptr = v
	}

	if err := app.InsertParameters(id, *fields[0].ptr, *fields[1].ptr, *fields[2].ptr, *fields[3].ptr, *fields[4].ptr, r.FormValue("notes")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tanks/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (app *App) createObservationHandler(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	note := r.FormValue("note")
	if note == "" {
		http.Error(w, "Observation note is required", http.StatusBadRequest)
		return
	}

	if err := app.InsertObservation(id, note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tanks/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (app *App) deleteTankHandler(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := app.DeleteTank(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
