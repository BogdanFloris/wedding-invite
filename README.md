# Wedding Invitation Website

A simple wedding invitation website built with Go, Templ, HTMX, and Tailwind CSS.

## Tech Stack

- **Backend**: Go
- **Frontend/Templates**: Templ + HTMX
- **Styling**: Tailwind CSS
- **Database**: SQLite
- **Deployment**: fly.io

## Development

### Prerequisites

- Go 1.21+ 
- templ CLI tool

### Setup

1. Install the templ CLI:
   ```bash
   go install github.com/a-h/templ/cmd/templ@latest
   ```

2. Generate templ files:
   ```bash
   templ generate
   ```

3. Run the server:
   ```bash
   go run cmd/server/main.go
   ```

4. Visit http://localhost:8080 in your browser.

## Deployment

Deploy to fly.io:

```bash
# Create the volume for storing SQLite database
fly volumes create wedding_data --size 1 --region otp

# Deploy the application
fly deploy
```

## Database

The application uses SQLite with a persistent volume in production. The database file is stored at `/data/wedding.db` in the container.