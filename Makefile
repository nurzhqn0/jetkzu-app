SHELL := /bin/bash

.PHONY: proto build test web-test docker-up docker-up-build docker-down docker-logs migrate-up migrate-down demo curl-demo

proto: ## Regenerate protobuf stubs (uses Docker, no local protoc needed)
	./scripts/gen-proto.sh

build: ## Build all binaries (inside Docker to avoid local Go dependency)
	docker compose build

docker-up: ## Start the entire stack
	docker compose up -d

docker-up-build: ## Rebuild images and start the entire stack
	docker compose up -d --build

docker-down: ## Stop the entire stack and remove volumes
	docker compose down -v

docker-logs: ## Tail logs from every service
	docker compose logs -f --tail=200

migrate-up: ## Apply all migrations against running Postgres
	docker compose run --rm migrate /bin/sh -c '\
	  for svc in users drivers rides payments notifications; do \
	    echo "==> Migrating jetkzu_$$svc"; \
	    migrate -path /migrations/$$svc -database "postgres://jetkzu:jetkzu@postgres:5432/jetkzu_$$svc?sslmode=disable" up; \
	  done'

migrate-down: ## Roll back all migrations (one step each)
	docker compose run --rm migrate /bin/sh -c '\
	  for svc in users drivers rides payments notifications; do \
	    echo "==> Rolling back jetkzu_$$svc"; \
	    migrate -path /migrations/$$svc -database "postgres://jetkzu:jetkzu@postgres:5432/jetkzu_$$svc?sslmode=disable" down 1 || true; \
	  done'

test: ## Run Go tests inside a Go image
	docker run --rm -v $$PWD:/src -w /src golang:1.23-alpine sh -c "go test ./..."

web-test: ## Build and test the frontend
	cd web && npm ci && npm run build && npm test

test-integration: ## Run integration tests (requires docker compose up)
	docker run --rm --network host -v $$PWD:/src -w /src \
	  -e RUN_INTEGRATION=1 -e GATEWAY_URL=http://localhost:8080 \
	  golang:1.23-alpine sh -c "go test ./tests/integration/..."

demo: ## Run the bundled curl demo flow against running gateway
	./scripts/demo.sh

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
