FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install swag for Swagger docs generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/services/wallet
# Generate Swagger docs
RUN swag init -g main.go --output docs --parseDependency --parseInternal || true
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/wallet-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/wallet-service .

EXPOSE 8005

CMD ["./wallet-service"]

