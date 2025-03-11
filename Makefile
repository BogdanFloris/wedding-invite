.PHONY: build run dev clean generate deploy add-invite

# Default target
all: generate build

# Generate templ templates
generate:
	@echo "Generating templates..."
	templ generate

# Build the application
build: generate
	@echo "Building application..."
	go build -o wedding-app ./cmd/server

# Run the application in development mode
dev: generate
	@echo "Starting development server..."
	go run cmd/server/main.go

# Run the application
run: build
	@echo "Starting server..."
	./wedding-app

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f wedding-app
	rm -f admin-tools
	rm -f *_templ.go
	find . -name "*_templ.go" -delete

# Deploy to fly.io
deploy: generate
	@echo "Deploying to fly.io..."
	fly deploy

# Add an invitation (helper tool)
add-invite:
	@echo "Running add invitation tool..."
	go run cmd/admin/add_invitation.go $(ARGS)

# Help for adding invitation
invite-help:
	@echo "Usage: make add-invite ARGS=\"-name 'Family Name' -guests 4 -email 'email@example.com' -phone '+1234567890'\""
	@echo ""
	@echo "Arguments:"
	@echo "  -name    Family name (required)"
	@echo "  -guests  Maximum number of guests (default: 2)"
	@echo "  -email   Contact email (optional)"
	@echo "  -phone   Contact phone (optional)"
	@echo "  -code    Custom invitation code (optional, generated if not provided)"