# UltraMedia Platform

An open source, modular video operations platform. It turns raw video files into structured, playable and observable media assets: upload, metadata extraction, thumbnails, HLS encoding, job logs and playback preview, with AI workflows planned on top.

> **Status:** early development. The foundation (repository, database, job queue, API and worker skeletons) is in place. Upload, FFmpeg processing and the dashboard screens are next. See the [Milestone 1 roadmap](docs/ultramedia-platform-roadmap-milestone-1.md).

## Architecture

```txt
SvelteKit dashboard ──► Go API ──► PostgreSQL (assets, jobs, logs)
                          │
                          ├──► Redis Streams ──► Go worker ──► FFmpeg / ffprobe
                          │                         │
                          └──────────► S3 storage ◄─┘  (sources and artifacts)
```

| Path | What it is |
|---|---|
| `apps/api` | HTTP API (Go) and database migrations |
| `apps/worker` | Job consumer (Go): reads the `jobs` stream and runs handlers |
| `apps/dashboard` | Web dashboard (SvelteKit) |
| `packages/job-contracts` | Job types, typed inputs and the queue message format |
| `packages/media-core` | Domain entities and the PostgreSQL connection pool |
| `infra/` | Local infrastructure config |
| `docs/` | PRD, roadmap and architecture decisions ([map](docs/README.md)) |
| `openspec/` | Specs and change proposals |

## Requirements

- Go 1.25+
- Node.js 22+ and npm
- Docker with Compose v2
- FFmpeg (needed once the real processing jobs land)

## Getting started

```sh
cp .env.example .env
make up        # PostgreSQL, Redis and S3 storage (RustFS); creates the bucket
make migrate   # applies database migrations
```

Then, in separate terminals:

```sh
make dev-api        # http://localhost:8080/healthz
make dev-worker
make dev-dashboard  # http://localhost:5173
```

The storage console is at http://localhost:9001 (user `ultramedia`, password `ultramedia`).

### Send a test job to the worker

Until the upload endpoint exists, you can push a job to the queue by hand:

```sh
docker compose exec redis redis-cli XADD jobs '*' \
  job_id 11111111-1111-1111-1111-111111111111 \
  type extract-metadata \
  payload '{"asset_id":"22222222-2222-2222-2222-222222222222","storage_key":"assets/demo/source/input.mp4"}'
```

The worker logs `stub: extract-metadata called for asset 2222…`.

## Common tasks

| Command | What it does |
|---|---|
| `make help` | List all targets |
| `make test` | `go vet` and unit tests (queue tests use Redis from `make up`, skipped if it is down) |
| `make build` | Build all Go modules |
| `make migrate-down` | Revert all migrations |
| `make down` | Stop infrastructure (data volumes are kept) |

## Configuration

All settings come from environment variables; see [`.env.example`](.env.example).

| Variable | Used by | Default |
|---|---|---|
| `DATABASE_URL` | API, migrations | required |
| `REDIS_URL` | worker | required |
| `API_PORT` | API | `8080` |
| `WORKER_NAME` | worker | hostname. Keep it stable so a restarted worker resumes unfinished jobs |
| `S3_ENDPOINT`, `S3_REGION`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET` | storage (upcoming) | see `.env.example` |
