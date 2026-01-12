FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install swag for Swagger docs generation
RUN go install github.com/swaggo/swag/cmd/swag@latest
ENV PATH="${PATH}:/root/go/bin"

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/services/post
# Generate Swagger docs
RUN swag init -g main.go --output docs --parseDependency --parseInternal || true
# Build with memory optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o /app/post-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/post-service .

EXPOSE 8002

CMD ["./post-service"]

