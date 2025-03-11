.PHONY: build run dev clean generate deploy

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
	rm -f *_templ.go
	find . -name "*_templ.go" -delete

# Deploy to fly.io
deploy: generate
	@echo "Deploying to fly.io..."
	fly deploy