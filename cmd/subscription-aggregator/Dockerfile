# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Копируем go.mod и go.sum из корня проекта
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники сервиса
COPY . .

# Собираем бинарь
RUN CGO_ENABLED=0 GOOS=linux go build -o subscription-aggregator ./cmd/subscription-aggregator/main.go

# Final stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/subscription-aggregator .
COPY --from=builder /app/config/config.yaml ./config.yaml

EXPOSE 8080

ENTRYPOINT ["./subscription-aggregator"]
