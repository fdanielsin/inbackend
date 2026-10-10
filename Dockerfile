FROM golang:1.26.9-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server .

FROM alpine:3.24.2

LABEL org.opencontainers.image.source="https://github.com/fdanielsin/inbackend" \
      org.opencontainers.image.title="inbackend" \
      org.opencontainers.image.description="Users and tasks HTTP API"

RUN apk --no-cache add ca-certificates \
    && adduser -D -u 10001 app

COPY --from=builder /server /usr/local/bin/server

USER app
EXPOSE 8080

HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=10 \
    CMD wget -qO /dev/null "http://localhost:${PORT:-8080}/ready"

CMD ["server"]
