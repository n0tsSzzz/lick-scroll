FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/services/feed
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/feed-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/feed-service .

EXPOSE 8003

CMD ["./feed-service"]

