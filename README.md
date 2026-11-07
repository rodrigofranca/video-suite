# UltraMedia Platform

An open source, modular video operations platform. It turns raw video files into structured, playable and observable media assets: upload, metadata extraction, thumbnails, HLS encoding, job logs and playback preview, with AI workflows planned on top.

> **Status:** early development. Upload → HLS encoding → playback works end to end in the dashboard, with job status and logs. Metadata extraction and thumbnails are still stubs (they log and succeed). See the [Milestone 1 roadmap](docs/ultramedia-platform-roadmap-milestone-1.md).

## Architecture

```txt
SvelteKit dashboard ──► Go API ──► PostgreSQL (assets, jobs, logs)
       │                  │
       │                  ├──► Redis Streams ──► Go worker ──► FFmpeg
       │                  │                         │
       │                  └──────────► S3 storage ◄─┘  (sources and HLS output)
       │                                   │
       └──── HLS playback (hls.js) ◄───────┘  only assets/*/hls/* is public
```

An upload creates an asset and three jobs: `extract-metadata` → `generate-thumbnail` → `transcode-hls`. The API queues the first; the worker runs each job, records its status and logs in PostgreSQL, and queues the next. The asset is `ready` when the HLS job succeeds.

| Path | What it is |
|---|---|
| `apps/api` | HTTP API (Go) and database migrations |
| `apps/worker` | Job runner (Go): reads the `jobs` stream, runs FFmpeg, chains the pipeline |
| `apps/dashboard` | Web dashboard (SvelteKit): upload, video list, job logs, player |
| `packages/job-contracts` | Job types, typed inputs, pipeline order and queue message format |
| `packages/media-core` | Domain entities, SQL (`store`), S3 client (`storage`), DB pool |
| `infra/` | Local infrastructure config (bucket CORS and public HLS policy) |
| `docs/` | PRD, roadmap and architecture decisions ([map](docs/README.md)) |
| `openspec/` | Specs and change proposals |

## Requirements

- Go 1.25+
- Node.js 22+ and npm
- Docker with Compose v2
- FFmpeg with libx264 (the worker refuses to start without it)

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

Open http://localhost:5173, click **Upload**, pick an MP4 and watch the jobs run. When the HLS job finishes, the video plays on the same page. The storage console is at http://localhost:9001 (user `ultramedia`, password `ultramedia`).

## API

| Method | Path | Returns |
|---|---|---|
| `POST` | `/assets/upload` | Upload (multipart field `file`, `.mp4`, up to 2 GB) → `201 {"assetId", "status"}` |
| `GET` | `/assets?status=` | Assets, newest first, optionally filtered by status |
| `GET` | `/assets/{id}` | Asset with source, jobs, artifacts and `playback` (when ready) |
| `GET` | `/assets/{id}/jobs` | Jobs in pipeline order |
| `GET` | `/assets/{id}/artifacts` | Files produced by jobs |
| `GET` | `/assets/{id}/playback` | `{"type": "hls", "manifestUrl"}`, or `409` while not ready |
| `GET` | `/jobs/{id}` | One job |
| `GET` | `/jobs/{id}/logs` | Job logs |
| `GET` | `/healthz` | Database check |

## Common tasks

| Command | What it does |
|---|---|
| `make help` | List all targets |
| `make test` | `go vet` and tests. Tests that need PostgreSQL, Redis, storage or FFmpeg are skipped when those are missing |
| `make build` | Build all Go modules |
| `make migrate-down` | Revert all migrations |
| `make down` | Stop infrastructure (data volumes are kept) |

## Configuration

All settings come from environment variables; see [`.env.example`](.env.example).

| Variable | Used by | Default |
|---|---|---|
| `DATABASE_URL` | API, worker, migrations | required |
| `REDIS_URL` | API, worker | required |
| `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET` | API, worker | required |
| `S3_REGION` | API, worker | `us-east-1` |
| `S3_PUBLIC_URL` | API (playback URLs) | `S3_ENDPOINT` |
| `API_PORT` | API | `8080` |
| `DASHBOARD_ORIGIN` | API (CORS) | `http://localhost:5173` |
| `PUBLIC_API_URL` | dashboard | `http://localhost:8080` |
| `WORKER_NAME` | worker | hostname. Keep it stable so a restarted worker resumes unfinished jobs |
| `WORKER_TEMP_DIR` | worker (FFmpeg work files) | system temp dir |
