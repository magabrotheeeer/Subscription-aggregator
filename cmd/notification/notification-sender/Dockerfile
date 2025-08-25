# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o notification-sender ./cmd/notification/notification-sender/main.go

# Final stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/notification-sender .
COPY --from=builder /app/config/config.yaml ./config.yaml

ENTRYPOINT ["./notification-sender"]
