.PHONY: up down init-db migrate migrate-status migrate-down migrate-docker seed-dev gen-proto gen-client-proto tidy test lint run-game-api run-auth run-game run-admin-api run-admin run-admin-web build-linux docker-build docker-up-prod

GOOS_LINUX ?= linux
GOARCH ?= amd64
BIN_DIR := bin
DATABASE_URL ?= postgres://game:game@localhost:5432/game?sslmode=disable
MIGRATE_DB ?= $(if $(MIGRATE_DATABASE_URL),$(MIGRATE_DATABASE_URL),$(DATABASE_URL))
COMPOSE := docker compose -f deploy/docker-compose.yml
COMPOSE_PROD := docker compose -f deploy/docker-compose.prod.yml
CLIENT_DIR := ../client
AUTH_DIR := ../auth-service
ADMIN_DIR := ../admin

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

# Prefer repo-root: make init-db / .\scripts\dev.ps1 init-db
init-db:
	bash ../deploy/postgresql/db-init.sh --write-env $(INIT_DB_ARGS)

migrate:
	go run ./cmd/migrate up

migrate-status:
	go run ./cmd/migrate status

migrate-down:
	go run ./cmd/migrate down $(or $(STEPS),1)

migrate-docker:
	docker run --rm -v "$(CURDIR)/migrations:/migrations" migrate/migrate \
		-path=/migrations -database "$(MIGRATE_DB)" up

seed-dev: migrate

run-admin-api:
	cd $(ADMIN_DIR) && go run ./cmd/admin-api

run-admin:
	cd $(ADMIN_DIR)/web && npm run build
	cd $(ADMIN_DIR) && go run ./cmd/admin-api

run-admin-web:
	cd $(ADMIN_DIR)/web && npm run dev

gen-proto:
	cd proto && buf generate

gen-client-proto:
	cd proto && buf generate --template buf.gen.client.yaml
	cd $(CLIENT_DIR) && node scripts/patch-proto-long.mjs

tidy:
	go mod tidy

test:
	go test ./...

lint:
	go run ./cmd/check-boundaries
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not installed; boundaries check only"; fi

run-game-api:
	go run ./cmd/game-api

run-auth:
	cd $(AUTH_DIR) && go run ./cmd/migrate -path migrations && go run ./cmd/auth-api

run-game:
	go run ./cmd/game

build-linux:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH) \
		go build -ldflags="-s -w" -o $(BIN_DIR)/game-api ./cmd/game-api
	CGO_ENABLED=0 GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH) \
		go build -ldflags="-s -w" -o $(BIN_DIR)/game ./cmd/game
	@echo "Built $(BIN_DIR)/game-api $(BIN_DIR)/game ($(GOOS_LINUX)/$(GOARCH))"

docker-build:
	docker build -f deploy/Dockerfile --target game-api -t game-api:latest .
	docker build -f deploy/Dockerfile --target game -t game-server:latest .

docker-up-prod:
	$(COMPOSE_PROD) up -d --build
