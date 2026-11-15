.PHONY: up down migrate migrate-status migrate-down migrate-docker seed-dev gen-proto gen-client-proto gen-client gen-client-api test lint test-client tidy run-game-api run-auth run-game run-admin-api run-admin run-admin-web build-linux docker-build docker-up-prod serve-bundles

SERVER := server

up down migrate migrate-status migrate-down migrate-docker seed-dev gen-proto tidy test lint run-game-api run-auth run-game run-admin-api run-admin run-admin-web build-linux docker-build docker-up-prod:
	@$(MAKE) -C $(SERVER) $@ STEPS=$(STEPS)

gen-client-proto:
	@$(MAKE) -C $(SERVER) gen-client-proto

gen-client-api:
	@echo ApiClient hand-written; skip OpenAPI codegen

gen-client: gen-client-proto gen-client-api

test-client:
	cd client && npm test

serve-bundles:
	@mkdir -p client/build/bundles
	@echo "Serving client/build/bundles on :8787 (requires python3)"
	cd client/build/bundles && python3 -m http.server 8787
