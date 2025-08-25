# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Копируем go.mod и go.sum из корня проекта
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники сервиса
COPY . .

# Собираем бинарь
RUN CGO_ENABLED=0 GOOS=linux go build -o auth ./cmd/auth/main.go

# Final stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/auth .
COPY --from=builder /app/config/config.yaml ./config.yaml
COPY --from=builder /app/migrations ./migrations

EXPOSE 50051

ENTRYPOINT ["./auth"]
