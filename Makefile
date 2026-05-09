.PHONY: run build test migrate

run:
	go run ./cmd/server

build:
	go build -o bin/stock-monitor ./cmd/server

test:
	go test ./internal/... -v

migrate:
	go run ./cmd/migrate
