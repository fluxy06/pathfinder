FROM golang:1.23.6-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o pathfinder ./cmd/pathfinder

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/pathfinder .
COPY .env .
CMD ["./pathfinder"]