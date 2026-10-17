# Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/seruwit-wa ./cmd/server

# Stage 2: Minimal runtime image
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/seruwit-wa /app/seruwit-wa
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/seruwit-wa"]
