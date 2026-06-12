package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "tyvm.db"
	}

	db, err := initDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	funcMap := template.FuncMap{
		"deref": func(p *float64) float64 {
			if p == nil {
				return 0
			}
			return *p
		},
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

	app := &App{
		db:        db,
		templates: tmpl,
	}

	http.HandleFunc("/", app.homeHandler)
	http.HandleFunc("/tanks/new", app.newTankHandler)
	http.HandleFunc("/tanks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			app.createTankHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	http.HandleFunc("/tanks/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/tanks/")
		parts := strings.Split(path, "/")

		if len(parts) >= 2 {
			if parts[1] == "log" {
				if r.Method == http.MethodPost {
					app.logParametersHandler(w, r)
				} else {
					app.logFormHandler(w, r)
				}
				return
			}
			if parts[1] == "observations" && r.Method == http.MethodPost {
				app.createObservationHandler(w, r)
				return
			}
		}

		app.tankDetailHandler(w, r)
	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
