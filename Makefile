.PHONY: up down init-db migrate migrate-status migrate-down migrate-docker seed-dev gen-proto gen-client-proto gen-client gen-client-api gen-jwt-keys test lint test-client tidy run-game-api run-auth run-game run-admin-api run-admin run-admin-web build-linux docker-build docker-up-prod serve-bundles

SERVER := server

up down migrate migrate-status migrate-down migrate-docker seed-dev gen-proto tidy test lint run-game-api run-auth run-game run-admin-api run-admin run-admin-web build-linux docker-build docker-up-prod:
	@$(MAKE) -C $(SERVER) $@ STEPS=$(STEPS)

# First-time role+DB bootstrap via deploy/postgresql. Examples:
#   make init-db
#   make init-db INIT_DB_ARGS='all --split-auth'
init-db:
	bash deploy/postgresql/db-init.sh --write-env $(INIT_DB_ARGS)

gen-client-proto:
	@$(MAKE) -C $(SERVER) gen-client-proto

gen-client-api:
	@echo ApiClient hand-written; skip OpenAPI codegen

gen-client: gen-client-proto gen-client-api

gen-jwt-keys:
	cd auth-service && go run ./keys/gen_jwt_keys.go
	@mkdir -p server/keys
	cp -f auth-service/keys/jwt-es256-dev.pub.pem server/keys/jwt-es256-dev.pub.pem
	@echo "ES256 keys in auth-service/keys/ (copy pub to server/keys/)"

test-client:
	cd client && npm test

serve-bundles:
	@mkdir -p client/build/bundles
	@echo "Serving client/build/bundles on :8787 (requires python3)"
	cd client/build/bundles && python3 -m http.server 8787
