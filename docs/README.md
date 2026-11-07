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
| Job order | all jobs created and run independently | `extract-metadata` → `generate-thumbnail` → `transcode-hls`, one at a time | `CONTEXT.md` |
| Tables | `assets`, `media_files`, `processing_jobs`, `job_logs` | `media_assets`, `media_sources`, `processing_jobs`, `job_logs`, `artifacts`, `technical_metadata` | migrations 0001–0007 |
| Upload title | `title` field | only `filename` for now | migration 0001 |
| Health route | `GET /health` | `GET /healthz` | `api-skeleton` spec |
| Local storage | MinIO | RustFS (S3 API) | ADR 0008 |
| Storage variables | `S3_*` (roadmap) / `MINIO_*` (old `.env`) | `S3_*` | `.env.example` |
| ADR numbering | ADR 001–004 inside the PRD | `docs/adr/0004`–`0007` | `docs/adr/` |

## Open decisions

Proposed but not yet confirmed. Each should become an ADR once decided.

- **Who enqueues the next job.** Proposal: the API creates all three job rows at upload (so the dashboard shows them as pending) but enqueues only the first; the worker enqueues the next one when a job ends, even if metadata or thumbnail failed.
- **How the browser reads HLS files.** Proposal: in development, make only `assets/*/hls/*` publicly readable in the bucket (CORS is already set up). Signed URLs are awkward for HLS because every segment listed in the playlist would need its own signature.
- **How the dashboard learns about status changes.** Proposal: poll every 2–3 seconds; consider server-sent events later.
- **Where the UltraMedia player comes from.** Start with `hls.js`, then plug in UltraMedia once it is packaged (npm package or web component).
