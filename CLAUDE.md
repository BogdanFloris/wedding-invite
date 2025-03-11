# Wedding Invitation Website - Developer Guide

## Tech Stack
- **Backend**: Go 1.24
- **Frontend**: Templ + HTMX
- **Styling**: Tailwind CSS (via CDN)
- **Database**: SQLite
- **Deployment**: fly.io

## Commands
- **Generate templates**: `make generate` or `templ generate`
- **Build**: `make build`
- **Run dev server**: `make dev`
- **Run production**: `make run`
- **Clean**: `make clean`
- **Deploy**: `make deploy`

## Code Style
- Go standard formatting (`gofmt`)
- Package structure: cmd/server (entry point), pkg/{db,handlers}, templates
- Error handling: Early returns with explicit error checks
- Database operations use prepared statements

## Development Flow
1. Edit .templ files in templates/
2. Run `make generate` to compile templates
3. Start dev server with `make dev`
4. Access at http://localhost:8080

## Dependencies
- github.com/a-h/templ v0.3.833
- github.com/mattn/go-sqlite3 v1.14.24