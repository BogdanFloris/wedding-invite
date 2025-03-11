# Wedding Invitation Website

A wedding invitation website with invitation code authentication, built with Go, Templ, HTMX, and Tailwind CSS.

## Features

- **Secure invitation code system**: Unique codes for each invited family
- **Direct link authentication**: Share URLs with embedded invitation codes
- **Mobile-first design**: Works great on all devices
- **Countdown timer**: Shows time until the wedding
- **Basic wedding details**: Schedule, venue, and event information
- **Security**: Rate limiting, CSRF protection, secure cookies

## Tech Stack

- **Backend**: Go 1.24
- **Frontend/Templates**: Templ + HTMX
- **Styling**: Tailwind CSS
- **Database**: SQLite
- **Deployment**: fly.io

## Development

### Prerequisites

- Go 1.24+ 
- templ CLI tool

### Setup

1. Install the templ CLI:
   ```bash
   go install github.com/a-h/templ/cmd/templ@latest
   ```

2. Generate templ files:
   ```bash
   make generate
   ```

3. Run the server:
   ```bash
   make dev
   ```

4. Visit http://localhost:8080 in your browser.

### Adding Invitations

Use the admin tool to add new invitation codes:

```bash
# See available options
make invite-help

# Add an invitation
make add-invite ARGS="-name 'Smith Family' -guests 4 -email 'smith@example.com' -phone '+1234567890'"
```

This generates a unique invitation code and URL that can be shared with guests.

## Deployment

### Environment Configuration

1. For local development, copy the example configuration:
   ```bash
   cp .env.example .env
   ```

2. Generate a secure SECRET_KEY:
   ```bash
   # Option 1: Using openssl
   openssl rand -base64 32
   
   # Option 2: Let the application generate one
   # Run the app once and it will print a suitable key to use
   ```

3. Edit `.env` to add your SECRET_KEY and customize other settings

### Deploy to fly.io

1. Initialize the application (first time only):
   ```bash
   fly launch --name wedding-invite --region otp --no-deploy
   ```

2. Create the volume for storing SQLite database:
   ```bash
   fly volumes create wedding_data --size 1 --region otp
   ```

3. Set the required secret for secure encryption:
   ```bash
   fly secrets set SECRET_KEY="your-generated-secret-key"
   ```

4. Deploy the application:
   ```bash
   fly deploy
   ```

## Authentication Flow

1. **Direct Link**: Guests visit `https://wedding.bogdanfloris.com/{invite-code}` and are authenticated automatically
2. **Manual Entry**: Alternatively, guests visit the home page and enter their invitation code
3. **Session**: Upon successful authentication, a secure session cookie is created
4. **Protected Content**: All wedding details are only visible to authenticated guests

## Security Considerations

- IP-based rate limiting (5 attempts per minute)
- CSRF protection on all forms
- Secure, HTTP-only cookies
- Password-free authentication
- IP addresses are hashed for privacy

## Database Schema

The application uses SQLite with tables for invitations, guests, and sessions.