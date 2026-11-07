# UltraMedia Platform — Domain Glossary

## Asset

A business entity representing a video uploaded by the user. Created once per upload, regardless of file content. Two uploads of the same file = two Assets.

An Asset has exactly one Source in the MVP.

Status is persisted in the database: `uploading → processing → ready | failed`. `uploading` lasts while the file streams to storage; an upload that fails or is rejected deletes the Asset instead of marking it `failed`. The API sets `processing` once the Source and Jobs are saved; the worker sets `ready` or `failed` when the HLS job ends. Dashboard filters and lists by status directly — no on-the-fly derivation.

`ready` = HLS generated and playable. Failures in metadata or thumbnail jobs are visible in Job logs but do not set Asset to `failed`. `failed` = HLS job failed.

**Not**: a physical file, a rendition, or a blob.

## Source

The original physical blob uploaded by the user. Tied 1:1 to an Asset in the MVP. Stored in S3-compatible storage under a generated key (never user-provided path). Never public: the metadata and thumbnail jobs read it through short-lived signed URLs, which are secrets and never logged or stored (ADR 0014).

**Not**: a rendition, a processed output, or an artifact.

## Rendition

A processed version of an Asset produced by a Job (e.g., HLS output, thumbnail). Stored as an Artifact.

In MVP: single HLS rendition only (no ABR). Multi-rendition is post-MVP.

**Not**: the original Source.

## Artifact

Any file output produced by a Job and associated with an Asset: the HLS playlist (`hls`) and the thumbnail image (`thumbnail`). Linked to both Asset and Job. HLS output and thumbnails are publicly readable (ADR 0010, 0013).

## Technical metadata

What ffprobe found in the Source, one row per Asset in `technical_metadata`: `{"summary": {...}, "ffprobe": {...}}`. The summary (duration, resolution, codecs, bitrate, frame rate, container, size) is what the dashboard shows; the full ffprobe output is kept for debugging.

## Job

A unit of processing work tied to an Asset. Has a type, input, status, logs, and timestamps. Execution model is opaque to the core — worker can be local, container, queue-based, or external.

Input is a typed Go struct defined in the `job-contracts` package, shared between API and Worker. API serializes to JSON for storage; Worker deserializes by job type. No dynamic schema validation in MVP.

A Job can be run again (`POST /assets/{id}/jobs`) once every Job of its Asset has finished. Each re-run is a new Job row — an attempt — so earlier attempts and their logs stay visible. A re-run runs only that Job; re-running `transcode-hls` puts the Asset back in `processing` (ADR 0015).

## Pipeline

Upload triggers automatic job pipeline: `extract-metadata` → `generate-thumbnail` → `transcode-hls` (order in `jobcontracts.Pipeline`). No manual trigger required. Worker processes one job at a time (no parallelism in MVP).

- `extract-metadata`: ffprobe → technical metadata.
- `generate-thumbnail`: a JPEG (max 1280 px wide) at `AtSecond`, or at the middle of the video when it is shorter; first frame as a last resort.
- `transcode-hls`: single-rendition HLS; reports progress (0–99 while encoding, 100 when done).

The API creates all three Job rows as `pending` and enqueues only the first. When a Job ends, done or failed, the worker enqueues the next pending Job of the same Asset (ADR 0009). Job statuses: `pending → running → done | failed`.

## Limits

MVP enforces: upload max **2GB**, job processing timeout **30 minutes**. API rejects oversized uploads before storage (by `Content-Length`, or while streaming when the size is unknown) and accepts only `.mp4` files that start with an MP4 `ftyp` box. Worker marks job as `failed` on timeout.

## Monorepo

Directory structure follows monorepo conventions (`apps/`, `packages/`) but no Turborepo/Nx in MVP. Makefile orchestrates builds and dev tasks. Turborepo added when JS packages multiply. Go modules handle `job-contracts` package sharing natively.

## Player

UltraMedia is the intended primary player in the dashboard, with `hls.js` as the fallback. Until UltraMedia is packaged, the dashboard uses `hls.js` (native HLS where MSE is missing), isolated in `HlsPlayer.svelte` (ADR 0012). No other player dependency in MVP.

The player reads HLS straight from storage: only `assets/*/hls/*` is publicly readable; sources stay private (ADR 0010).

## Worker

Process that picks pending Jobs and executes them. Separate from the API process. In MVP: single worker binary, same machine.

Queue: Redis Streams (XADD/XREADGROUP) for at-least-once delivery and job re-claim on worker crash. NATS is a future upgrade path.

Queue message: fields `job_id`, `type` and `payload` (JSON of the typed input), defined in `packages/job-contracts`. The PostgreSQL job row is the source of truth; the message only says which job to run. The worker acks every message, even when the job fails. It does not ack a job interrupted by shutdown (the job stays `running`), and on restart it first re-runs messages it received but never acked, so `WORKER_NAME` must stay stable. A duplicate delivery of a finished job is skipped.
