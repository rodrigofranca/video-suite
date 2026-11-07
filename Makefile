.PHONY: help up down demo demo-logs sample sample-speech llm-check migrate migrate-down dev-api dev-worker dev-ai-worker dev-dashboard build test test-go test-ai env-check

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

up: ## Start infrastructure (postgres, redis, storage, LiteLLM gateway) and create the bucket
	docker compose up -d --wait postgres redis storage litellm
	docker compose run --rm storage-init

down: ## Stop and remove all containers, including the demo stack (data volumes are kept)
	docker compose --profile app down

demo: ## Build and start everything in Docker; open http://localhost:5173
	docker compose --profile app up -d --build --wait
	@echo "UltraMedia is running: dashboard http://localhost:5173, API http://localhost:8080"

demo-logs: ## Follow the API and worker logs of the demo stack
	docker compose --profile app logs -f api worker ai-worker litellm

sample: ## Create samples/sample.mp4 (30 s, 720p, with sound) with the worker image's FFmpeg
	@mkdir -p samples
	docker compose --profile app build worker
	docker run --rm --user "$$(id -u):$$(id -g)" -v "$(CURDIR)/samples:/out" --entrypoint ffmpeg \
		ultramedia/worker:dev -hide_banner -loglevel error -y \
		-f lavfi -i "testsrc2=duration=30:size=1280x720:rate=30" \
		-f lavfi -i "sine=frequency=440:duration=30" \
		-c:v libx264 -preset veryfast -c:a aac -shortest /out/sample.mp4
	@echo "Created samples/sample.mp4"

sample-speech: ## Create samples/speech.mp4 with spoken English, to try transcription (local FFmpeg with flite)
	@mkdir -p samples
	@ffmpeg -hide_banner -filters 2>/dev/null | grep -q " flite " || \
		(echo "Your FFmpeg has no flite filter (Ubuntu and Debian builds do). Use any MP4 with speech instead." && exit 1)
	ffmpeg -hide_banner -loglevel error -y \
		-f lavfi -i "testsrc2=duration=30:size=1280x720:rate=30" \
		-f lavfi -i "flite=voice=slt:text='Welcome to the ultra media platform. This video was uploaded, encoded and transcribed automatically. The transcript and the subtitles were created by an A I worker running Whisper on this machine.'" \
		-c:v libx264 -preset veryfast -c:a aac -ar 44100 -shortest samples/speech.mp4
	@echo "Created samples/speech.mp4"

llm-check: ## Ask the language model one short question through the gateway (checks OPENCODE_API_KEY)
	@curl -sS --max-time 60 http://localhost:4000/v1/chat/completions \
		-H "Authorization: Bearer $${LITELLM_MASTER_KEY:-sk-ultramedia-dev}" -H "Content-Type: application/json" \
		-d '{"model": "ultramedia-text", "max_tokens": 20, "messages": [{"role": "user", "content": "Reply with: OK"}]}' \
		| python3 -c 'import json, sys; d = json.load(sys.stdin); print(d["choices"][0]["message"]["content"] if "choices" in d else d)'

migrate: env-check ## Apply all pending database migrations
	@$(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down: env-check ## Revert all database migrations
	@$(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" down -all

dev-api: env-check ## Start API development server on port 8080
	@go run ./apps/api/cmd/api

dev-worker: env-check ## Start worker development process
	@go run ./apps/worker/cmd/worker

dev-ai-worker: env-check ## Start the AI worker (Python, needs uv); downloads the Whisper model on first run
	@cd apps/ai-worker && uv run python -m ultramedia_ai

dev-dashboard: ## Start SvelteKit dashboard dev server on port 5173
	@cd apps/dashboard && (test -d node_modules || npm install) && npm run dev

build: ## Build all Go modules
	go build $(GO_PACKAGES)

test: test-go test-ai ## Run all tests (Go and Python)

test-go: ## Run Go vet and tests
	go vet $(GO_PACKAGES)
	go test $(GO_PACKAGES)

test-ai: ## Run the AI worker tests (Python, needs uv; transcription tests download the tiny model)
	cd apps/ai-worker && uv run pytest -q

env-check:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is not set. Run: cp .env.example .env" && exit 1)
