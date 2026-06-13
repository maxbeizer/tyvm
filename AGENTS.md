# AGENTS.md

## What This Is

**tyvm** ("tank you very much") — a lightweight self-hosted aquarium tracker.
Go + SQLite + plain HTML templates. Mobile-first PWA. Single binary.

## Stack

- **Language:** Go 1.22+
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **Templates:** Go `html/template` in `templates/`
- **Static assets:** `static/` (CSS, PWA manifest, service worker)
- **Router:** stdlib `net/http` only — no framework

## Project Structure

```
tyvm/
├── main.go          # HTTP server, Go 1.22 method+path routing
├── models.go        # Domain types
├── db.go            # Schema + all SQL queries (no SQL in handlers)
├── handlers.go      # Thin request handlers
├── csrf.go          # Double-submit-cookie CSRF middleware
├── sparkline.go     # Inline-SVG sparkline rendering
├── *_test.go        # Unit/integration tests
├── templates/       # HTML templates (forms carry hidden _csrf field)
│   ├── new_tank.html
│   ├── index.html
│   ├── tank.html
│   └── log.html
├── static/
│   ├── style.css
│   ├── manifest.json
│   └── sw.js
├── Dockerfile
├── go.mod
└── README.md
```

## Running Locally

```bash
go run .
# → http://localhost:8080
```

## Building

```bash
go build -o tyvm .
./tyvm
```

## Docker

```bash
docker build -t tyvm .
docker run -p 8080:8080 -v $(pwd)/data:/app/data -e DB_PATH=/app/data/tyvm.db tyvm
```

## Environment Variables

| Var | Default | Description |
|-----|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `tyvm.db` | SQLite database file path |

## Database

Three tables: `tanks`, `parameters`, `observations`. Schema in `db.go`.
SQLite file is the entire database — back it up by copying the file.

## Conventions

- Keep dependencies minimal — stdlib first, only add external deps when necessary
- No JS frameworks — server-rendered HTML only
- Mobile-first CSS — design for 375px, scale up
- **All SQL lives in `db.go`** — handlers call methods on `*App`, never run raw SQL
- Routing uses Go 1.22 method+path patterns (`GET /tanks/{id}/log`); extract path vars with `r.PathValue`
- All state-changing POSTs are protected by `csrfMiddleware`; templates rendering forms must include `<input type="hidden" name="_csrf" value="{{.CSRFToken}}">`
- Templates use `base.html` as layout wrapper

## Adding a Feature

1. Add DB query function to `db.go` if needed
2. Add handler to `handlers.go`
3. Register route in `main.go`
4. Add/update template in `templates/`

## Git Identity

```
git config user.name "Claudia"
git config user.email "heyclaudia@users.noreply.github.com"
```

Push auth: `https://heyclaudia:$(cat ~/.gh-token-claudia)@github.com/maxbeizer/tyvm.git`
