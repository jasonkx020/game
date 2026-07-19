.PHONY: up down migrate migrate-status migrate-down migrate-docker seed-dev gen-proto test run-api run-game run-admin tidy build-linux docker-build docker-up-prod serve-bundles

GOOS_LINUX ?= linux
GOARCH ?= amd64
BIN_DIR := bin
DATABASE_URL ?= postgres://game:game@localhost:5432/game?sslmode=disable
MIGRATE_DB ?= $(if $(MIGRATE_DATABASE_URL),$(MIGRATE_DATABASE_URL),$(DATABASE_URL))
COMPOSE := docker compose -f deploy/docker-compose.yml
COMPOSE_PROD := docker compose -f deploy/docker-compose.prod.yml

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

# 本地 / CI：仅需 Go，读取 .env DATABASE_URL
migrate:
	go run ./cmd/migrate up

migrate-status:
	go run ./cmd/migrate status

migrate-down:
	go run ./cmd/migrate down $(or $(STEPS),1)

# 可选：Docker 内执行（生产 compose 仍用 migrate/migrate 镜像）
migrate-docker:
	docker run --rm -v "$(CURDIR)/migrations:/migrations" migrate/migrate \
		-path=/migrations -database "$(MIGRATE_DB)" up

seed-dev: migrate

run-admin:
	cd web/admin && npm run dev

gen-proto:
	cd proto && buf generate

gen-client-proto:
	cd proto && buf generate --template buf.gen.client.yaml
	cd client && node scripts/patch-proto-long.mjs

gen-client-api:
	@echo ApiClient hand-written; skip OpenAPI codegen

gen-client: gen-client-proto gen-client-api

test-client:
	cd client && npm test

tidy:
	go mod tidy

test:
	go test ./...

run-api:
	go run ./cmd/platform-api

run-game:
	go run ./cmd/game

# 交叉编译 Linux 二进制（在 Windows/macOS 开发机上执行，产物部署到 Linux）
build-linux:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH) \
		go build -ldflags="-s -w" -o $(BIN_DIR)/platform-api ./cmd/platform-api
	CGO_ENABLED=0 GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH) \
		go build -ldflags="-s -w" -o $(BIN_DIR)/game ./cmd/game
	@echo "Built $(BIN_DIR)/platform-api $(BIN_DIR)/game ($(GOOS_LINUX)/$(GOARCH))"

docker-build:
	docker build -f deploy/Dockerfile --target platform-api -t game-platform-api:latest .
	docker build -f deploy/Dockerfile --target game -t game-server:latest .

docker-up-prod:
	$(COMPOSE_PROD) up -d --build

serve-bundles:
	@mkdir -p client/build/bundles
	@echo "Serving client/build/bundles on :8787 (requires python3)"
	cd client/build/bundles && python3 -m http.server 8787
