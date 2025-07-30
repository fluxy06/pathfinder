# Этап сборки
FROM golang:1.23.6-bullseye AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o pathfinder ./cmd/pathfinder

# Финальный образ
FROM debian:bullseye-slim
WORKDIR /app
COPY --from=builder /app/pathfinder .
CMD ["./pathfinder"]