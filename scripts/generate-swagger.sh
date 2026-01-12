#!/bin/bash

# Script to generate Swagger documentation for all services

export PATH=$PATH:$(go env GOPATH)/bin

SERVICES=("auth" "post" "feed" "fanout" "wallet" "notification" "moderation" "analytics")

for service in "${SERVICES[@]}"; do
    echo "Generating Swagger docs for $service service..."
    cd "services/$service"
    if [ -f "main.go" ]; then
        swag init -g main.go --output docs --parseDependency --parseInternal 2>&1 | grep -E "(error|warning|create)" || echo "✓ $service docs generated"
    else
        echo "⚠ main.go not found in services/$service"
    fi
    cd ../..
done

echo "Swagger documentation generation complete!"

