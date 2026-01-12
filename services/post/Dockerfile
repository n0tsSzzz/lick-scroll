FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/services/post
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/post-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/post-service .

EXPOSE 8002

CMD ["./post-service"]

