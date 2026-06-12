# 🐟 tyvm

**tank you very much** — A minimal aquarium tracker

Track water parameters, observations, and maintenance for your aquariums. Mobile-first PWA built with Go + SQLite.

## Features

- 📊 Log water parameters (pH, ammonia, nitrite, nitrate, temperature)
- 📝 Track observations and notes
- 🐠 Manage multiple tanks
- 📱 Mobile-first responsive design
- 💾 Single SQLite database file
- 🚀 Single binary deployment

## Quick Start

### Run with Docker

```bash
docker build -t tyvm .
docker run -p 8080:8080 -v $(pwd)/data:/data -e DB_PATH=/data/tyvm.db tyvm
```

Visit http://localhost:8080

### Run locally

```bash
# Install dependencies
go mod download

# Run the app
go run .

# Or build and run
go build -o tyvm .
./tyvm
```

## Environment Variables

- `PORT` — Server port (default: `8080`)
- `DB_PATH` — SQLite database path (default: `tyvm.db`)

## Stack

- **Backend:** Go 1.22+ with standard library
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **Frontend:** Plain HTML templates + vanilla CSS
- **PWA:** Service worker + manifest for offline support

## Database Schema

### Tables

- `tanks` — Tank information (name, size, type, notes)
- `parameters` — Water parameter logs (pH, ammonia, nitrite, nitrate, temp)
- `observations` — General observations and notes

## Development

The app uses Go's `html/template` for rendering and standard library HTTP server. No external frameworks required.

Project structure:
- `main.go` — HTTP server and routing
- `db.go` — Database initialization and schema
- `handlers.go` — Request handlers
- `templates/` — HTML templates
- `static/` — CSS, JS, PWA assets

## License

MIT
