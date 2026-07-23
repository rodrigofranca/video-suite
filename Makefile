.PHONY: up down migrate migrate-down dev-api dev-worker dev-dashboard help

include .env 2>/dev/null || true
export

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

up: ## Start all infrastructure services (postgres, redis, minio)
	DOCKER_CONFIG=/tmp/dockercfg docker compose up -d 2>/dev/null || docker compose up -d

down: ## Stop and remove all infrastructure containers
	docker compose down

migrate: ## Apply all pending database migrations
	@migrate -path apps/api/migrations -database "$(DATABASE_URL)" up

migrate-down: ## Revert all database migrations
	@migrate -path apps/api/migrations -database "$(DATABASE_URL)" down $$(migrate -path apps/api/migrations -database "$(DATABASE_URL)" version 2>/dev/null | awk '{print $$1}')

dev-api: ## Start API development server on port 8080
	@go run ./apps/api/cmd/api

dev-worker: ## Start worker development process
	@go run ./apps/worker/cmd/worker

dev-dashboard: ## Start SvelteKit dashboard dev server on port 5173
	@cd apps/dashboard && npm run dev
