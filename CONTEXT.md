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

Any file output produced by a Job and associated with an Asset: the HLS playlist (`hls`), the thumbnail image (`thumbnail`), the extracted audio (`audio`), subtitles (`subtitles`) and exported clips (`clip`). Linked to both Asset and Job. HLS output, thumbnails and subtitles are publicly readable (ADR 0010, 0013, 0017); clips are not (ADR 0020).

## Technical metadata

What ffprobe found in the Source, one row per Asset in `technical_metadata`: `{"summary": {...}, "ffprobe": {...}}`. The summary (duration, resolution, codecs, bitrate, frame rate, container, size) is what the dashboard shows; the full ffprobe output is kept for debugging.

## Job

A unit of processing work tied to an Asset. Has a type, input, status, logs, and timestamps. Execution model is opaque to the core — worker can be local, container, queue-based, or external.

Input is a typed Go struct defined in the `job-contracts` package, shared between API and Worker. API serializes to JSON for storage; Worker deserializes by job type. No dynamic schema validation in MVP.

A Job can be run again (`POST /assets/{id}/jobs`) once every Job of its Asset has finished. Each re-run is a new Job row — an attempt — so earlier attempts and their logs stay visible. A re-run runs only that Job; re-running `transcode-hls` puts the Asset back in `processing` (ADR 0015).

## Pipeline

Upload triggers automatic job pipeline: `extract-metadata` → `generate-thumbnail` → `transcode-hls` → `extract-audio` → `transcribe-audio` → `analyze-transcript` → `suggest-clips` (order in `jobcontracts.Pipeline`). No manual trigger required. Each worker processes one job at a time (no parallelism in MVP).

- `extract-metadata`: ffprobe → technical metadata.
- `generate-thumbnail`: a JPEG (max 1280 px wide) at `AtSecond`, or at the middle of the video when it is shorter; first frame as a last resort.
- `transcode-hls`: single-rendition HLS; reports progress (0–99 while encoding, 100 when done). The Asset becomes `ready` here; the next steps run while it is already playable (ADR 0017).
- `extract-audio`: the first audio track as 16 kHz mono FLAC, private. A video without audio is not an error.
- `transcribe-audio`: Whisper (AI worker) → Transcript and WebVTT subtitles.
- `analyze-transcript`: a language model, through the LLM gateway (AI worker) → Analysis. Nothing to do when there is no speech.
- `suggest-clips`: the language model picks Clips from the transcript (AI worker). Nothing to do under 30 seconds of speech.

`export-clip` is not in the pipeline: it runs when someone asks for a Clip in a format, and, like a re-run, only when no other Job of the Asset is pending or running.

The API creates all the Job rows as `pending` and enqueues only the first. When a Job ends, done or failed, the worker enqueues the next pending Job of the same Asset (ADR 0009). Job statuses: `pending → running → done | failed`.

## Transcript

What was said in an Asset, written by the `transcribe-audio` Job: language (detected unless given), full text, segments with start and end times, every word with its start and end (for word-by-word captions; empty for transcripts made before they were saved), and the Whisper model used. One per Asset in `transcripts`; a re-run replaces it. Subtitles are built from it.

## Analysis

What a language model wrote about an Asset from its Transcript, by the `analyze-transcript` Job: a summary, topic tags and chapters (start in seconds and a title), in the Transcript's language, plus the model that answered. One per Asset in `analyses`; a re-run replaces it (ADR 0019). Chapters always start where a transcript passage starts.

**Not**: technical metadata (ffprobe) or the metadata `summary` object, which describe the file, not its content.

## Clip

A moment of an Asset that works as a short standalone video, suggested by the `suggest-clips` Job: start and end on transcript segment boundaries (10 to 90 seconds), title, reason and a score from 1 to 10. Up to 5 per Asset in `clips`; a new suggestion run replaces them. An export (`export-clip`) cuts a Clip into an MP4 in one format, `original`, `vertical` (centre crop, 9:16) or `vertical-blur` (whole frame over a blurred copy, 9:16), optionally with burned-in captions in one of five styles: `highlight` (the word being said in yellow, the default), `box` (a box behind it), `karaoke` (words fill with color as they are said), `one-word` (only the word being said, big) — these need the Transcript's word timings (ADR 0021) — or `lines` (subtitle lines). Exported files are private and downloaded through signed links (ADR 0020).

**Not**: a Rendition (which is the whole video for playback) or a chapter (which marks a part of the video without cutting it).

## LLM gateway

The LiteLLM service every language model call goes through (ADR 0018). Services use the OpenAI API with a model name defined in the gateway (`ultramedia-text`); the provider (OpenCode first), its model ID and its key exist only in `infra/litellm/config.yaml` and the gateway's environment.

## Limits

MVP enforces: upload max **2GB**, job processing timeout **30 minutes**. API rejects oversized uploads before storage (by `Content-Length`, or while streaming when the size is unknown) and accepts only `.mp4` files that start with an MP4 `ftyp` box. Worker marks job as `failed` on timeout.

## Monorepo

Directory structure follows monorepo conventions (`apps/`, `packages/`) but no Turborepo/Nx in MVP. Makefile orchestrates builds and dev tasks. Turborepo added when JS packages multiply. Go modules handle `job-contracts` package sharing natively.

## Player

UltraMedia is the intended primary player in the dashboard, with `hls.js` as the fallback. Until UltraMedia is packaged, the dashboard uses `hls.js` (native HLS where MSE is missing), isolated in `HlsPlayer.svelte` (ADR 0012). No other player dependency in MVP.

The player reads HLS straight from storage: only `assets/*/hls/*`, `assets/*/thumbnails/*` and `assets/*/subtitles/*` are publicly readable; sources and extracted audio stay private (ADR 0010, 0013, 0017). Subtitles are sidecar WebVTT `<track>` files.

## Worker

Process that picks pending Jobs and executes them. Separate from the API process. Two kinds, each reading its own stream (ADR 0016):

- **Media worker** (Go, `apps/worker`): stream `jobs`, group `workers`; FFmpeg jobs, including `export-clip`.
- **AI worker** (Python, `apps/ai-worker`): stream `jobs:ai`, group `ai-workers`; `transcribe-audio`, `analyze-transcript` and `suggest-clips`. It mirrors the media worker's lifecycle, and its contracts are checked against `packages/job-contracts/testdata/contracts.json`.

`jobcontracts.StreamFor` decides which stream a Job goes to.

Queue: Redis Streams (XADD/XREADGROUP) for at-least-once delivery and job re-claim on worker crash. NATS is a future upgrade path.

Queue message: fields `job_id`, `type` and `payload` (JSON of the typed input), defined in `packages/job-contracts`. The PostgreSQL job row is the source of truth; the message only says which job to run. The worker acks every message, even when the job fails. It does not ack a job interrupted by shutdown (the job stays `running`), and on restart it first re-runs messages it received but never acked, so `WORKER_NAME` (and `AI_WORKER_NAME`) must stay stable. A duplicate delivery of a finished job is skipped.
