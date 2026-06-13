package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
)

type App struct {
	db        *sql.DB
	templates *template.Template
}

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

	app := &App{db: db, templates: tmpl}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", app.homeHandler)
	mux.HandleFunc("GET /tanks/new", app.newTankHandler)
	mux.HandleFunc("POST /tanks", app.createTankHandler)
	mux.HandleFunc("GET /tanks/{id}", app.tankDetailHandler)
	mux.HandleFunc("GET /tanks/{id}/log", app.logFormHandler)
	mux.HandleFunc("POST /tanks/{id}/log", app.logParametersHandler)
	mux.HandleFunc("POST /tanks/{id}/observations", app.createObservationHandler)
	mux.HandleFunc("POST /tanks/{id}/delete", app.deleteTankHandler)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, csrfMiddleware(mux)))
}
