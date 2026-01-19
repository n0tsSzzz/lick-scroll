#!/bin/bash

echo "🧪 Running all tests with coverage..."
echo ""

# Run tests with coverage (including all packages)
go test -coverprofile=coverage.out ./pkg/jwt/... ./services/auth/handlers/... ./services/auth/repository/... ./services/post/internal/controller/http/... ./services/post/repository/... ./services/notification/internal/controller/http/... ./pkg/middleware/... ./pkg/config/... ./pkg/logger/... ./pkg/models/...

echo ""
echo "📊 Coverage by package:"
go test -coverprofile=coverage.out ./pkg/jwt/... ./services/auth/handlers/... ./services/auth/repository/... ./services/post/internal/controller/http/... ./services/post/repository/... ./services/notification/internal/controller/http/... ./pkg/middleware/... ./pkg/config/... ./pkg/logger/... ./pkg/models/... | grep "coverage:"

echo ""
echo "📈 Overall coverage:"
go tool cover -func=coverage.out | tail -1

echo ""
echo "📋 Test count:"
go test -v ./pkg/jwt/... ./services/auth/handlers/... ./services/auth/repository/... ./services/post/internal/controller/http/... ./services/post/repository/... ./services/notification/internal/controller/http/... ./pkg/middleware/... ./pkg/config/... ./pkg/logger/... ./pkg/models/... 2>&1 | grep -E "^=== RUN" | wc -l | xargs echo "Total tests:"
