.PHONY: help up down demo demo-logs sample migrate migrate-down dev-api dev-worker dev-dashboard build test env-check

# Load local settings (copy .env.example to .env first).
-include .env
export

GO_PACKAGES := ./apps/api/... ./apps/worker/... ./packages/job-contracts/... ./packages/media-core/...
MIGRATIONS  := apps/api/migrations
# Runs the golang-migrate CLI without installing it globally.
MIGRATE     := go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

help: ## Show available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

up: ## Start infrastructure (postgres, redis, storage) and create the bucket
	docker compose up -d --wait postgres redis storage
	docker compose run --rm storage-init

down: ## Stop and remove all containers, including the demo stack (data volumes are kept)
	docker compose --profile app down

demo: ## Build and start everything in Docker; open http://localhost:5173
	docker compose --profile app up -d --build --wait
	@echo "UltraMedia is running: dashboard http://localhost:5173, API http://localhost:8080"

demo-logs: ## Follow the API and worker logs of the demo stack
	docker compose --profile app logs -f api worker

sample: ## Create samples/sample.mp4 (30 s, 720p, with sound) with the worker image's FFmpeg
	@mkdir -p samples
	docker compose --profile app build worker
	docker run --rm --user "$$(id -u):$$(id -g)" -v "$(CURDIR)/samples:/out" --entrypoint ffmpeg \
		ultramedia/worker:dev -hide_banner -loglevel error -y \
		-f lavfi -i "testsrc2=duration=30:size=1280x720:rate=30" \
		-f lavfi -i "sine=frequency=440:duration=30" \
		-c:v libx264 -preset veryfast -c:a aac -shortest /out/sample.mp4
	@echo "Created samples/sample.mp4"

migrate: env-check ## Apply all pending database migrations
	@$(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down: env-check ## Revert all database migrations
	@$(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" down -all

dev-api: env-check ## Start API development server on port 8080
	@go run ./apps/api/cmd/api

dev-worker: env-check ## Start worker development process
	@go run ./apps/worker/cmd/worker

dev-dashboard: ## Start SvelteKit dashboard dev server on port 5173
	@cd apps/dashboard && (test -d node_modules || npm install) && npm run dev

build: ## Build all Go modules
	go build $(GO_PACKAGES)

test: ## Run Go vet and unit tests
	go vet $(GO_PACKAGES)
	go test $(GO_PACKAGES)

env-check:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is not set. Run: cp .env.example .env" && exit 1)
