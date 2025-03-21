FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install required system packages
RUN apk add --no-cache gcc musl-dev

# Copy go.mod and go.sum files
COPY go.mod go.sum* ./
RUN go mod download

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy source code
COPY . .

# Generate templ files
RUN templ generate

# Build main app and admin tool
RUN CGO_ENABLED=1 go build -o wedding-app ./cmd/server && \
    CGO_ENABLED=1 go build -o admin-tool ./cmd/admin

FROM alpine:3.19

WORKDIR /app

# Install required runtime dependencies
RUN apk add --no-cache ca-certificates sqlite

# Copy the binaries from builder
COPY --from=builder /app/wedding-app .
COPY --from=builder /app/admin-tool .

# Copy static and template files
COPY --from=builder /app/static ./static
COPY --from=builder /app/locales ./locales

# Copy and set permissions for admin script
COPY admin.sh .
RUN chmod +x admin.sh

# Create volume for database
VOLUME /data

# Set environment variables
ENV PORT=8080
ENV DB_PATH=/data/wedding.db

EXPOSE 8080

CMD ["./wedding-app"]
