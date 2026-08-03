FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/api \
    && CGO_ENABLED=0 go build -ldflags="-s -w" -o worker ./cmd/worker

FROM builder AS goose-build
RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.26.0

# Одноразовый контейнер миграций: goose + каталог migrations.
# Запускается compose-сервисом migrate до старта приложения.
FROM alpine:3.22 AS migrator
RUN adduser -D -u 10001 appuser
COPY --from=goose-build /go/bin/goose /usr/local/bin/goose
COPY migrations /migrations
USER appuser
ENTRYPOINT ["goose"]
CMD ["up"]

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /app/server /app/worker ./
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1
CMD ["./server"]
