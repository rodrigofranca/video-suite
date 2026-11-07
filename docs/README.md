# Documentation map

## Which document wins

When two documents disagree, the one higher on this list is correct:

1. **[`CONTEXT.md`](../CONTEXT.md)**: domain glossary and current rules (statuses, pipeline, limits).
2. **[`docs/adr/`](adr/)**: architecture decisions, one per file.
3. **[`openspec/specs/`](../openspec/specs/)**: what the code is required to do today. Changes in progress live in `openspec/changes/`.
4. **[PRD v0.1](ultramedia-platform-prd-v0.1.md)** and **[Milestone 1 roadmap](ultramedia-platform-roadmap-milestone-1.md)**: product vision, scope and plan. Their technical details are older than items 1–3.
5. **[`docs/discovery/`](discovery/)**: the original brainstorming and a first PRD draft. Kept for history only.

## Known differences in the PRD and roadmap

These parts of the PRD v0.1 and the roadmap are superseded:

| Topic | PRD / roadmap say | Current rule | Source |
|---|---|---|---|
| Asset statuses | `created`, `processing`, `ready`, `failed` | `uploading`, `processing`, `ready`, `failed` | `CONTEXT.md`, migration 0001 |
| Job statuses | `completed` | `done` | migration 0003, `media-core/domain` |
| When an asset is `ready` | all jobs succeeded | the HLS job succeeded | ADR 0003 |
| Job order | all jobs created and run independently | `extract-metadata` → `generate-thumbnail` → `transcode-hls` → `extract-audio` → `transcribe-audio` → `analyze-transcript` → `suggest-clips`, one at a time | `CONTEXT.md` |
| Tables | `assets`, `media_files`, `processing_jobs`, `job_logs` | `media_assets`, `media_sources`, `processing_jobs`, `job_logs`, `artifacts`, `technical_metadata`, `transcripts`, `analyses`, `clips` | migrations 0001–0011 |
| Upload title | `title` field | only `filename` for now | migration 0001 |
| Health route | `GET /health` | `GET /healthz` | `api-skeleton` spec |
| Local storage | MinIO | RustFS (S3 API) | ADR 0008 |
| Storage variables | `S3_*` (roadmap) / `MINIO_*` (old `.env`) | `S3_*` | `.env.example` |
| ADR numbering | ADR 001–004 inside the PRD | `docs/adr/0004`–`0007` | `docs/adr/` |
| HLS playlist name | `master.m3u8` | `index.m3u8` (single rendition) | `storage.HLSManifestKey` |
| Playback API | `GET /assets/:id/playback` only | also included in `GET /assets/:id` as `playback` | `asset-api` spec |

## Decisions taken in step 2

Confirmed on 2026-11-07 and recorded as ADRs:

- [0009](adr/0009-pipeline-chaining.md): the API creates all job rows and enqueues the first; the worker enqueues the next when a job ends.
- [0010](adr/0010-public-hls-playback.md): the player reads HLS straight from storage; only `assets/*/hls/*` is public.
- [0011](adr/0011-status-polling.md): the dashboard polls every 3 seconds.
- [0012](adr/0012-hls-js-before-ultramedia.md): `hls.js` until the UltraMedia player is packaged.

## Decisions taken in step 3

- [0013](adr/0013-public-thumbnails.md): thumbnails are public, like the HLS output.
- [0014](adr/0014-signed-urls-for-source-reads.md): the metadata and thumbnail jobs read the source through signed URLs instead of downloading it.

## Decisions taken in Milestone 2

- [0016](adr/0016-python-ai-worker-own-stream.md): the AI worker is Python, reads its own `jobs:ai` stream and mirrors the Go contracts.
- [0017](adr/0017-transcription-after-hls.md): transcription runs after HLS; transcripts in the database, public WebVTT subtitles.
- [0018](adr/0018-llm-gateway-litellm.md): language models only through a LiteLLM gateway; OpenCode is the first provider; the image is pinned by digest.
- [0019](adr/0019-transcript-analysis.md): one analysis per asset after transcription: summary, tags and chapters from one checked JSON answer.

## Decisions taken in the AI clipping phase

- [0020](adr/0020-ai-clips.md): the model suggests clips in the pipeline; exports run on request in three formats, with optional burned-in captions, and stay private behind signed download links.
- [0021](adr/0021-word-by-word-captions.md): transcripts keep each word's timing; clip captions default to short phrases with the spoken word highlighted.
