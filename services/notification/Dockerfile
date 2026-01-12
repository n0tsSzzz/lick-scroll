FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/services/notification
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/notification-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/notification-service .

EXPOSE 8006

CMD ["./notification-service"]

