.PHONY: help build run test clean generate migrate-up migrate-down swagger sqlc install-tools deps setup docker-up docker-down docker-build

# Load environment variables
include .env

# Default target
help:
	@echo "Available targets:"
	@echo "  install-tools - Install required development tools"
	@echo "  setup        - Setup development environment"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  generate     - Generate code from OpenAPI and SQLC"
	@echo "  migrate-up   - Run database migrations up"
	@echo "  migrate-down - Run database migrations down"
	@echo "  migrate-create - Create a new migration"
	@echo "  swagger      - Generate Swagger documentation"
	@echo "  sqlc         - Generate SQLC code"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  lint-fix     - Lint and fix code automatically"
	@echo "  docker-up    - Start PostgreSQL with Docker Compose"
	@echo "  docker-down  - Stop Docker Compose services"
	@echo "  docker-build - Build Docker image"

# Build the application
build:
	go build -o bin/server cmd/server/main.go

# Run the application
run:
	go run cmd/server/main.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Generate all code
generate: swagger sqlc

# Install required development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Tools installed successfully!"

# Generate Swagger documentation
swagger:
	~/go/bin/swag init -g cmd/server/main.go -o ./docs

# Generate SQLC code
sqlc:
	~/go/bin/sqlc generate

# Run database migrations up
migrate-up:
	~/go/bin/goose -dir migrations postgres $(DB_DSN) up

# Run database migrations down
migrate-down:
	~/go/bin/goose -dir migrations postgres $(DB_DSN) down

# Create a new migration
migrate-create:
	@read -p "Enter migration name: " name; \
	~/go/bin/goose -dir migrations create $$name sql

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

# Lint and fix code
lint-fix: fmt
	@echo "Running linters..."
	@make lint

# Docker Compose commands
docker-up:
	docker-compose up -d postgres

docker-down:
	docker-compose down

docker-build:
	docker-compose build

# Setup development environment
setup: install-tools deps docker-up
	sleep 5
	make migrate-up
	make generate
