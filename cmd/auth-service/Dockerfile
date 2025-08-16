# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Копируем go.mod и go.sum из корня проекта
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники сервиса
COPY ./cmd/auth-service ./cmd/auth-service
COPY ./config/config.yaml ./config/config.yaml

# Собираем бинарь
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd/auth-service/main.go

# Final stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/auth-service .
COPY --from=builder /app/config/config.yaml ./config.yaml

EXPOSE 50051

ENTRYPOINT ["./auth-service"]
