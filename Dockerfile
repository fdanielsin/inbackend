FROM golang:1.26.8-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server .

FROM alpine:3.24.2

RUN apk --no-cache add ca-certificates \
    && adduser -D -u 10001 app

COPY --from=builder /server /usr/local/bin/server

USER app
EXPOSE 8080

CMD ["server"]
