.PHONY: help dev build clean test docker-up docker-down

help: ## Show this help message
	@echo "Uptime Kuma (Go + Next.js) - Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

setup: ## Initial setup (install dependencies)
	@echo "Setting up Go dependencies..."
	go mod download
	@echo "Setting up Next.js dependencies..."
	cd web && npm install
	@echo "Creating data directory..."
	mkdir -p data
	@echo "Setup complete!"

dev: ## Run development servers (Go + Next.js)
	@echo "Starting development servers..."
	@make -j2 dev-backend dev-frontend

dev-backend: ## Run Go backend in development mode
	@echo "Starting Go backend on :8080..."
	go run cmd/server/main.go

dev-frontend: ## Run Next.js frontend in development mode
	@echo "Starting Next.js frontend on :3000..."
	cd web && npm run dev

build: ## Build production binaries
	@echo "Building Go backend..."
	go build -o uptime-kuma-go cmd/server/main.go
	@echo "Building Next.js frontend..."
	cd web && npm run build
	@echo "Build complete!"

clean: ## Clean build artifacts and data
	@echo "Cleaning build artifacts..."
	rm -f uptime-kuma-go
	rm -rf web/.next web/out web/build
	rm -rf data/*.db data/*.db-shm data/*.db-wal
	@echo "Clean complete!"

test: ## Run all tests
	@echo "Running Go tests..."
	go test -v ./...
	@echo "Running Next.js tests..."
	cd web && npm test

test-backend: ## Run Go backend tests
	go test -v ./...

lint: ## Run linters
	@echo "Linting Go code..."
	go vet ./...
	@echo "Linting Next.js code..."
	cd web && npm run lint

docker-up: ## Start Docker development environment
	docker-compose up -d

docker-down: ## Stop Docker development environment
	docker-compose down

docker-build: ## Build Docker image
	docker build -t uptime-kuma-go:latest .

run: ## Run production build
	./uptime-kuma-go
