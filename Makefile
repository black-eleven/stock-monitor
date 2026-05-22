.PHONY: run build test migrate docker-build docker-up docker-down docker-restart docker-logs

run:
	go run ./cmd/server

build:
	go build -o bin/stock-monitor ./cmd/server

test:
	go test ./internal/... -v

migrate:
	go run ./cmd/migrate

docker-build:
	docker build -t stock-monitor:latest .

docker-save:
	docker save -o stock-monitor.tar stock-monitor:latest

docker-upload: docker-build docker-save
	scp stock-monitor.tar root@114.55.60.62:/root/workspace/src/github.com/black-eleven/stock-monitor

docker-load:
	docker load -i stock-monitor.tar

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-restart:
	docker compose restart

docker-logs:
	docker compose logs -f
