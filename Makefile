.PHONY: build build-all test clean docker-build docker-up docker-down

# Build all services
build-all:
	@echo "Building all services..."
	@cd services/auth && go build -o ../../bin/auth-service .
	@cd services/post && go build -o ../../bin/post-service .
	@cd services/feed && go build -o ../../bin/feed-service .
	@cd services/fanout && go build -o ../../bin/fanout-service .
	@cd services/wallet && go build -o ../../bin/wallet-service .
	@cd services/notification && go build -o ../../bin/notification-service .
	@cd services/analytics && go build -o ../../bin/analytics-service .
	@echo "Build complete!"

# Build specific service
build:
	@echo "Building service: $(SERVICE)"
	@cd services/$(SERVICE) && go build -o ../../bin/$(SERVICE)-service .

# Run tests
test:
	@echo "Running tests..."
	@go test ./pkg/jwt/... ./services/auth/internal/controller/http/... ./services/post/internal/controller/http/... ./services/notification/internal/controller/http/... ./pkg/middleware/... ./pkg/config/... ./pkg/logger/... ./pkg/models/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./pkg/jwt/... ./services/auth/internal/controller/http/... ./services/post/internal/controller/http/... ./services/notification/internal/controller/http/... ./pkg/middleware/... ./pkg/config/... ./pkg/logger/... ./pkg/models/...
	@echo ""
	@echo "Coverage report:"
	@go tool cover -func=coverage.out | tail -10

# Run tests with verbose output
test-v:
	@echo "Running tests with verbose output..."
	@go test -v ./pkg/jwt/... ./services/auth/internal/controller/http/... ./services/post/internal/controller/http/... ./services/notification/internal/controller/http/... ./pkg/middleware/... ./pkg/config/... ./pkg/logger/... ./pkg/models/...

# Show coverage summary
coverage:
	@go test -coverprofile=coverage.out ./pkg/jwt/... ./services/auth/internal/controller/http/... ./services/post/internal/controller/http/... ./services/notification/internal/controller/http/... ./pkg/middleware/... ./pkg/config/... ./pkg/logger/... ./pkg/models/...
	@echo ""
	@echo "📊 Coverage by package:"
	@go test -coverprofile=coverage.out ./pkg/jwt/... ./services/auth/internal/controller/http/... ./services/post/internal/controller/http/... ./services/notification/internal/controller/http/... ./pkg/middleware/... ./pkg/config/... ./pkg/logger/... ./pkg/models/... | grep "coverage:"
	@echo ""
	@echo "📈 Overall coverage:"
	@go tool cover -func=coverage.out | tail -1

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean

# Docker commands
docker-build:
	@echo "Building Docker images..."
	@docker-compose build

docker-up:
	@echo "Starting services..."
	@docker-compose up -d

docker-down:
	@echo "Stopping services..."
	@docker-compose down

docker-logs:
	@docker-compose logs -f

# Run service locally (requires postgres and redis running)
run-auth:
	@cd services/auth && go run main.go

run-post:
	@cd services/post && go run main.go

run-feed:
	@cd services/feed && go run main.go

run-fanout:
	@cd services/fanout && go run main.go

run-wallet:
	@cd services/wallet && go run main.go

run-notification:
	@cd services/notification && go run main.go


run-analytics:
	@cd services/analytics && go run main.go

# Database migrations (using goose)
migrate:
	@echo "Running migrations..."
	@go run ./cmd/migrate/main.go -dir=migrations -command=up

migrate-down:
	@echo "Rolling back migrations..."
	@go run ./cmd/migrate/main.go -dir=migrations -command=down

migrate-status:
	@echo "Migration status:"
	@go run ./cmd/migrate/main.go -dir=migrations -command=status

migrate-create:
	@echo "Creating new migration: $(NAME)"
	@go run ./cmd/migrate/main.go -dir=migrations -command=create -name=$(NAME)

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

