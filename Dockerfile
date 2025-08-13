# стабильная версия golang; 1.23.6 может отсутствовать
FROM golang:1.22-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o pathfinder ./cmd/pathfinder

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/pathfinder .
# данные монтируются из compose
ENTRYPOINT ["./pathfinder"]
