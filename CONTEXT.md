# UltraMedia Platform — Domain Glossary

## Asset

A business entity representing a video uploaded by the user. Created once per upload, regardless of file content. Two uploads of the same file = two Assets.

An Asset has exactly one Source in the MVP.

Status is persisted in the database: `uploading → processing → ready | failed`. Worker updates status after each Job. Dashboard filters and lists by status directly — no on-the-fly derivation.

`ready` = HLS generated and playable. Failures in metadata or thumbnail jobs are visible in Job logs but do not set Asset to `failed`. `failed` = HLS job failed.

**Not**: a physical file, a rendition, or a blob.

## Source

The original physical blob uploaded by the user. Tied 1:1 to an Asset in the MVP. Stored in S3-compatible storage under a generated key (never user-provided path).

**Not**: a rendition, a processed output, or an artifact.

## Rendition

A processed version of an Asset produced by a Job (e.g., HLS output, thumbnail). Stored as an Artifact.

In MVP: single HLS rendition only (no ABR). Multi-rendition is post-MVP.

**Not**: the original Source.

## Artifact

Any file output produced by a Job and associated with an Asset (e.g., HLS manifest, thumbnail image, ffprobe JSON). Linked to both Asset and Job.

## Job

A unit of processing work tied to an Asset. Has a type, input, status, logs, and timestamps. Execution model is opaque to the core — worker can be local, container, queue-based, or external.

Input is a typed Go struct defined in the `job-contracts` package, shared between API and Worker. API serializes to JSON for storage; Worker deserializes by job type. No dynamic schema validation in MVP.

## Pipeline

Upload triggers automatic job pipeline: `extract-metadata` → `generate-thumbnail` → `transcode-hls`. No manual trigger required. Worker processes one job at a time (no parallelism in MVP). Jobs are queued sequentially per upload.

## Limits

MVP enforces: upload max **2GB**, job processing timeout **30 minutes**. API rejects oversized uploads before storage. Worker marks job as `failed` on timeout.

## Monorepo

Directory structure follows monorepo conventions (`apps/`, `packages/`) but no Turborepo/Nx in MVP. Makefile orchestrates builds and dev tasks. Turborepo added when JS packages multiply. Go modules handle `job-contracts` package sharing natively.

## Player

UltraMedia is the primary player in the dashboard. `hls.js` is the fallback for environments where UltraMedia is not available or fails to load. No other player dependency in MVP.

## Worker

Process that picks pending Jobs and executes them. Separate from the API process. In MVP: single worker binary, same machine.

Queue: Redis Streams (XADD/XREADGROUP) for at-least-once delivery and job re-claim on worker crash. NATS is a future upgrade path.

Queue message: fields `job_id`, `type` and `payload` (JSON of the typed input), defined in `packages/job-contracts`. The PostgreSQL job row is the source of truth; the message only says which job to run. The worker acks every message, even when the job fails. On restart it first re-runs messages it received but never acked, so `WORKER_NAME` must stay stable.
