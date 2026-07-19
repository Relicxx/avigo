.PHONY: build run test test-race vet fmt lint tidy

build:
	go build ./...

run:
	go run ./cmd/api

test:
	go test ./...

test-race:
	go test -race -cover ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run

tidy:
	go mod tidy
