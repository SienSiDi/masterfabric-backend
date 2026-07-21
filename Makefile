.PHONY: build run test test-cover lint migrate migrate-down docker-up docker-down clean security-scan

build:
	go build -o bin/masterfabric-server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

test-cover:
	go test -cover ./...

lint:
	golangci-lint run

migrate:
	goose -dir internal/infrastructure/postgres/migrations postgres "$(DATABASE_DSN)" up

migrate-down:
	goose -dir internal/infrastructure/postgres/migrations postgres "$(DATABASE_DSN)" down

docker-up:
	docker compose -f deployments/docker-compose.yml up -d

docker-down:
	docker compose -f deployments/docker-compose.yml down

clean:
	rm -rf bin/

security-scan:
	govulncheck ./...
	gosec -quiet ./...
