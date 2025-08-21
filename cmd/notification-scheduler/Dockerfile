# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o notification-scheduler ./cmd/notification-scheduler/main.go

# Final stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/notification-scheduler .
COPY --from=builder /app/config/config.yaml ./config.yaml

ENTRYPOINT ["./notification-scheduler"]

