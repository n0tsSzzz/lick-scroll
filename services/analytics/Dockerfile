FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install swag for Swagger docs generation
RUN go install github.com/swaggo/swag/cmd/swag@latest
ENV PATH="${PATH}:/root/go/bin"

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/services/analytics
# Generate Swagger docs
RUN swag init -g main.go --output docs --parseDependency --parseInternal || true
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/analytics-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/analytics-service .

EXPOSE 8008

CMD ["./analytics-service"]

