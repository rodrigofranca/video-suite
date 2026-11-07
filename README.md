# UltraMedia Platform

An open source, modular video operations platform. It turns raw video files into structured, playable and observable media assets: upload, metadata extraction, thumbnails, HLS encoding, job logs and playback preview, with AI workflows planned on top.

> **Status:** Milestone 1 (video processing core) is complete: upload, technical metadata (ffprobe), thumbnail, HLS encoding with progress, job logs, re-runs and playback, runnable with one command. Next is Milestone 2 (AI metadata). See the [roadmap](docs/ultramedia-platform-roadmap-milestone-1.md) and the [known limitations](#known-limitations).

## Architecture

```txt
SvelteKit dashboard ──► Go API ──► PostgreSQL (assets, jobs, logs)
       │                  │
       │                  ├──► Redis Streams ──► Go worker ──► FFmpeg
       │                  │                         │
       │                  └──────────► S3 storage ◄─┘  (sources and HLS output)
       │                                   │
       └── HLS playback, thumbnails ◄──────┘  only hls/ and thumbnails/ are public
```

An upload creates an asset and three jobs: `extract-metadata` → `generate-thumbnail` → `transcode-hls`. The API queues the first; the worker runs each job, records its status, progress and logs in PostgreSQL, and queues the next. The asset is `ready` when the HLS job succeeds. Metadata and thumbnail jobs read the source through signed URLs (only the bytes they need); the HLS job downloads it.

| Path | What it is |
|---|---|
| `apps/api` | HTTP API (Go) and database migrations |
| `apps/worker` | Job runner (Go): reads the `jobs` stream, runs FFmpeg, chains the pipeline |
| `apps/dashboard` | Web dashboard (SvelteKit): upload, video list, job logs, player |
| `packages/job-contracts` | Job types, typed inputs, pipeline order and queue message format |
| `packages/media-core` | Domain entities, SQL (`store`), S3 client (`storage`), DB pool |
| `infra/` | Local infrastructure config (bucket CORS and public-read policy) |
| `docs/` | PRD, roadmap and architecture decisions ([map](docs/README.md)) |
| `openspec/` | Specs and change proposals |

## Quick start (Docker only)

```sh
make demo     # builds and starts everything; first build takes a few minutes
make sample   # optional: creates samples/sample.mp4 to upload
```

Open http://localhost:5173 and upload an MP4. `make demo-logs` follows the API and worker logs; `make down` stops everything. A walkthrough for presenting it is in [docs/demo.md](docs/demo.md).

## Development

### Requirements

- Go 1.25+
- Node.js 22+ and npm
- Docker with Compose v2
- FFmpeg with libx264, including `ffprobe` (the worker refuses to start without them)

### Running from source

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

Open http://localhost:5173, click **Upload**, pick an MP4 and watch the jobs run: metadata and thumbnail take a moment, then HLS encoding shows its progress. When it finishes, the video plays on the same page. The storage console is at http://localhost:9001 (user `ultramedia`, password `ultramedia`).

## API

| Method | Path | Returns |
|---|---|---|
| `POST` | `/assets/upload` | Upload (multipart field `file`, `.mp4`, up to 2 GB) → `201 {"assetId", "status"}` |
| `GET` | `/assets?status=` | Assets, newest first, with `thumbnailUrl` and `metadata` summary |
| `GET` | `/assets/{id}` | Asset with source, jobs, artifacts, `thumbnailUrl`, `metadata` and `playback` (when ready) |
| `GET` | `/assets/{id}/metadata` | Metadata summary and full ffprobe output |
| `GET` | `/assets/{id}/jobs` | Jobs in pipeline order, with `progress` (0–100) |
| `GET` | `/assets/{id}/artifacts` | Files produced by jobs |
| `GET` | `/assets/{id}/playback` | `{"type": "hls", "manifestUrl"}`, or `409` while not ready |
| `POST` | `/assets/{id}/jobs` | Run one job again: `{"type": "generate-thumbnail"}` → `202` with the new job; `409` while any job of the asset is pending or running |
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
| `make demo` / `make demo-logs` | Run everything in Docker / follow its API and worker logs |
| `make sample` | Create `samples/sample.mp4` with the worker image's FFmpeg |
| `make down` | Stop all containers, including the demo stack (data volumes are kept) |

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

## Known limitations

Milestone 1 is a working core, not a production service:

- **No authentication.** Anyone who can reach the API can upload. HLS output and thumbnails are public to anyone with the URL; the demo uses fixed development credentials.
- **One rendition.** HLS is a single H.264/AAC rendition in MPEG-TS segments, no adaptive bitrate ladder.
- **MP4 only, up to 2 GB.** Files must start with an MP4 `ftyp` box; MP4s that begin with another box are rejected.
- **One job at a time.** A single worker runs jobs in order, so a long encode delays the next upload. Jobs have a 30-minute limit.
- **No automatic retries.** A failed job stays failed until someone runs it again from the dashboard or the API. A re-run runs only that job.
- **Crash recovery needs a stable worker name.** A restarted worker resumes its own unfinished job only if `WORKER_NAME` is unchanged.
- **Progress starts after the download.** The encoding percentage stays at 0 while the worker downloads the source.
- **Leftover segments.** Re-running HLS overwrites the playlist and segments, but if the new run produces fewer segments the extra old files stay in storage (the playlist does not reference them).
- **No deletion.** Assets cannot be deleted from the API or dashboard yet.
- **The dashboard image is built for one API address.** `PUBLIC_API_URL` is fixed when the image is built (`http://localhost:8080` in the demo).
