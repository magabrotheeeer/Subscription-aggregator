# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o payment-processor ./cmd/payment-processor/main.go

# Final stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/payment-processor .
COPY --from=builder /app/config/config.yaml ./config.yaml

ENTRYPOINT ["./payment-processor"]
