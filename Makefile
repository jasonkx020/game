.PHONY: up down migrate migrate-down seed-dev gen-proto test run-api run-game run-admin tidy

up:
	docker compose -f deploy/docker-compose.yml up -d

down:
	docker compose -f deploy/docker-compose.yml down

migrate:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

seed-dev: migrate
	migrate -path migrations -database "$(DATABASE_URL)" up

run-admin:
	cd web/admin && npm run dev

gen-proto:
	cd proto && buf generate

gen-client-proto:
	cd proto && buf generate --template buf.gen.client.yaml

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
