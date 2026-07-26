FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/api \
    && CGO_ENABLED=0 go build -ldflags="-s -w" -o worker ./cmd/worker

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /app/server /app/worker ./
COPY --from=builder /app/migrations ./migrations
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1
CMD ["./server"]
