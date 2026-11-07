# PRD — UltraMedia Platform

**Version:** 0.1  
**Status:** Draft  
**Project:** UltraMedia Platform  
**Scope:** MVP 1 — Video Processing Core  
**Owner:** Rodrigo França  
**Last updated:** 2026-05-08  

> **Note (2026-11-07):** this PRD still defines the vision and scope, but some technical details are out of date (statuses, tables, job order, storage server). Where it disagrees with `CONTEXT.md`, `docs/adr/` or `openspec/specs/`, those win. The differences are listed in [`docs/README.md`](README.md).

---

# 1. Overview

UltraMedia Platform is an open source, modular, AI-first video engineering suite for ingesting, processing, enriching, monitoring, and delivering video content.

The platform is designed as a living monorepo where each feature builds on top of a shared media core. The initial focus is VOD processing: upload, metadata extraction, thumbnail generation, HLS encoding, artifact storage, job monitoring, and playback preview.

Over time, the platform may evolve into a complete video operations environment, including AI-assisted NOC, automatic social clipping, live streaming, restreaming, cuepoint generation, object detection, and extensible processing workflows.

---

# 2. Product Vision

Build an open source Video Ops platform that turns raw video files into structured, enriched, playable, and observable media assets.

The platform should help engineers, creators, and video teams understand and control the complete lifecycle of a video asset:

```txt
Upload
  ↓
Ingest
  ↓
Extract metadata
  ↓
Process
  ↓
Encode
  ↓
Enrich with AI
  ↓
Store artifacts
  ↓
Preview
  ↓
Monitor
  ↓
Deliver
```

The project should also act as a practical lab for advanced video engineering, combining:

```txt
FFmpeg
ffprobe
HLS
DASH
media metadata
job processing
AI workflows
video observability
playback integration
developer automation
```

---

# 3. Problem Statement

Video engineering workflows are usually fragmented.

A single video pipeline may involve multiple disconnected tools and services:

```txt
upload tools
FFmpeg scripts
metadata extraction
manual subtitle generation
cloud encoders
storage buckets
custom dashboards
logs in different places
separate playback tools
manual debugging
```

This creates problems:

- Hard to understand the full lifecycle of a video.
- Hard to debug failed encoding jobs.
- Hard to inspect generated artifacts.
- Hard to connect metadata, jobs, logs, and playback.
- Hard to experiment with AI workflows.
- Hard to evolve from VOD to live and clipping workflows.

UltraMedia Platform aims to solve this by creating a modular and observable video processing foundation.

---

# 4. Goals

## 4.1 Product Goals

The first version should allow a user to:

1. Upload an MP4 file.
2. Create a media asset.
3. Extract technical metadata using `ffprobe`.
4. Generate a thumbnail using `FFmpeg`.
5. Transcode the video to HLS.
6. Store all generated artifacts.
7. Track job status and logs.
8. Preview the generated HLS video in a dashboard.

## 4.2 Technical Goals

The first version should establish:

1. A shared media domain model.
2. A job-based processing architecture.
3. A storage abstraction compatible with S3.
4. A local development environment with Docker Compose.
5. A clean separation between API, worker, dashboard, and shared packages.
6. A foundation for future AI workers.
7. A foundation for future observability and NOC features.

## 4.3 Career / Portfolio Goals

The project should demonstrate strong knowledge of:

```txt
video processing
backend architecture
worker queues
FFmpeg
HLS packaging
media metadata
SvelteKit dashboards
Go APIs
storage systems
AI-first workflows
open source engineering
```

---

# 5. Non-Goals

The following features are explicitly out of scope for MVP 1:

```txt
live streaming
RTMP ingest
SRT ingest
WebRTC ingest
restreaming
automatic clipping
object detection
face tracking
AI NOC
multi-tenant SaaS
billing
DRM
ad insertion
advanced analytics
plugin marketplace
Kubernetes deployment
```

These features may be added later, but only after the Video Processing Core is stable.

---

# 6. Target Users

## 6.1 Primary User

A video engineer or developer who wants to process and inspect videos locally or in a self-hosted environment.

Needs:

- Upload videos.
- Run processing jobs.
- Inspect metadata.
- Generate HLS output.
- Preview results.
- Debug failures.

## 6.2 Secondary User

A technical creator or indie hacker who wants to build AI-powered video workflows.

Needs:

- Generate subtitles.
- Extract highlights.
- Create clips.
- Generate metadata.
- Automate content workflows.

## 6.3 Future User

A video operations team that wants an open source NOC-like system for media pipelines.

Needs:

- Monitor job health.
- Diagnose failures.
- Inspect assets.
- Analyze playback issues.
- Use AI to explain incidents.

---

# 7. MVP 1 Scope — Video Processing Core

## 7.1 Main Flow

```txt
User uploads MP4
  ↓
API stores source file
  ↓
API creates Asset
  ↓
API creates Processing Jobs
  ↓
Worker extracts metadata
  ↓
Worker generates thumbnail
  ↓
Worker transcodes video to HLS
  ↓
Worker stores artifacts
  ↓
Dashboard shows progress
  ↓
User previews HLS video
```

## 7.2 Required Features

### Feature 1 — Upload Video

User can upload an MP4 video through the dashboard.

Requirements:

- Accept `.mp4` files.
- Store original file in object storage.
- Create a `MediaFile` record with kind `source`.
- Create an `Asset` record.
- Trigger initial processing jobs.

Acceptance criteria:

- User can upload a valid MP4.
- Source file is stored successfully.
- Asset appears in the dashboard.
- Asset starts with status `processing`.

---

### Feature 2 — Asset Management

The platform should manage uploaded videos as media assets.

Requirements:

- Create asset after upload.
- List assets.
- Show asset details.
- Show asset status.
- Link asset to source file and generated artifacts.

Acceptance criteria:

- Dashboard lists all assets.
- User can open an asset detail page.
- Asset detail page shows title, status, metadata, files, jobs, and preview when available.

---

### Feature 3 — Metadata Extraction

The worker should extract technical metadata using `ffprobe`.

Metadata to extract:

```txt
duration
width
height
video codec
audio codec
bitrate
frame rate
container format
file size
```

Acceptance criteria:

- Metadata job is created after upload.
- Worker runs `ffprobe`.
- Metadata is persisted.
- Dashboard displays metadata.
- Failed metadata extraction updates job status to `failed`.

---

### Feature 4 — Thumbnail Generation

The worker should generate at least one thumbnail using `FFmpeg`.

Requirements:

- Generate a thumbnail from the uploaded video.
- Store thumbnail in object storage.
- Create `MediaFile` record with kind `thumbnail`.
- Link thumbnail to the asset.

Acceptance criteria:

- Thumbnail job is created after upload.
- Worker generates image successfully.
- Thumbnail appears in the asset list and asset detail page.
- Failed thumbnail generation updates job status to `failed`.

---

### Feature 5 — HLS Encoding

The worker should transcode the uploaded MP4 into HLS.

Initial HLS output:

```txt
master.m3u8
media playlist
.ts or .m4s segments
```

Requirements:

- Generate HLS output using FFmpeg.
- Store manifest and segments in object storage.
- Create `MediaFile` record with kind `manifest`.
- Link generated HLS output to the asset.

Acceptance criteria:

- HLS job is created after upload.
- Worker generates HLS output successfully.
- HLS manifest is accessible.
- Asset status becomes `ready` after successful processing.
- Dashboard can preview the HLS video.

---

### Feature 6 — Processing Jobs

All heavy video operations should run as jobs.

Initial job types:

```txt
extract-metadata
generate-thumbnail
transcode-hls
```

Job statuses:

```txt
pending
running
completed
failed
```

Acceptance criteria:

- Each processing step creates or updates a job.
- Jobs have visible statuses.
- Jobs have timestamps.
- Failed jobs store error messages.
- Dashboard shows job history per asset.

---

### Feature 7 — Job Logs

Workers should persist logs for each processing job.

Log levels:

```txt
info
warning
error
```

Acceptance criteria:

- Logs are linked to jobs.
- Dashboard shows logs per job.
- FFmpeg or ffprobe errors are captured.
- Logs help debug failed jobs.

---

### Feature 8 — Dashboard Preview

User can preview the generated HLS video in the dashboard.

Requirements:

- Asset detail page includes a player.
- Player loads the generated HLS manifest.
- If HLS is not ready, preview area shows processing state.

Acceptance criteria:

- User can open a ready asset.
- User can play generated HLS output.
- If video is still processing, user sees current status.

---

# 8. User Experience

## 8.1 Dashboard Navigation

Initial dashboard pages:

```txt
/assets
/assets/:id
/upload
/jobs
```

## 8.2 Asset List Page

The asset list should show:

```txt
thumbnail
title
status
duration
resolution
created date
processing state
```

## 8.3 Asset Detail Page

The asset detail page should show:

```txt
video preview
thumbnail
technical metadata
source file
generated files
processing jobs
job logs
HLS manifest URL
```

## 8.4 Upload Page

The upload page should allow:

```txt
select file
set title
start upload
see upload progress
redirect to asset detail
```

---

# 9. Domain Model

## 9.1 Asset

Represents a video inside the platform.

```ts
type Asset = {
  id: string;
  title: string;
  status: "created" | "processing" | "ready" | "failed";
  sourceFileId?: string;
  duration?: number;
  width?: number;
  height?: number;
  bitrate?: number;
  videoCodec?: string;
  audioCodec?: string;
  frameRate?: number;
  containerFormat?: string;
  createdAt: string;
  updatedAt: string;
};
```

## 9.2 MediaFile

Represents a physical file or object in storage.

```ts
type MediaFile = {
  id: string;
  assetId: string;
  kind:
    | "source"
    | "thumbnail"
    | "rendition"
    | "manifest"
    | "subtitle"
    | "clip";

  path: string;
  bucket: string;
  mimeType: string;
  size: number;
  createdAt: string;
};
```

## 9.3 ProcessingJob

Represents an asynchronous media processing operation.

```ts
type ProcessingJob = {
  id: string;
  assetId: string;
  type:
    | "extract-metadata"
    | "generate-thumbnail"
    | "transcode-hls";

  status: "pending" | "running" | "completed" | "failed";
  input: Record<string, unknown>;
  output?: Record<string, unknown>;
  error?: string;
  progress?: number;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
};
```

## 9.4 JobLog

Represents logs generated by workers.

```ts
type JobLog = {
  id: string;
  jobId: string;
  level: "info" | "warning" | "error";
  message: string;
  createdAt: string;
};
```

---

# 10. API Requirements

## 10.1 Assets

### `GET /assets`

Returns all assets.

### `GET /assets/:id`

Returns asset details, including files, jobs, and metadata.

### `POST /assets/upload`

Uploads a video and creates an asset.

Initial request:

```txt
multipart/form-data
file: video.mp4
title: string
```

Response:

```json
{
  "assetId": "asset_123",
  "status": "processing"
}
```

---

## 10.2 Jobs

### `GET /jobs`

Returns recent jobs.

### `GET /jobs/:id`

Returns job details.

### `GET /assets/:id/jobs`

Returns all jobs for an asset.

### `GET /jobs/:id/logs`

Returns logs for a job.

---

## 10.3 Files

### `GET /assets/:id/files`

Returns all files for an asset.

### `GET /assets/:id/playback`

Returns playback information.

Example response:

```json
{
  "type": "hls",
  "manifestUrl": "http://localhost:9000/ultramedia/assets/asset_123/hls/master.m3u8"
}
```

---

# 11. Worker Requirements

## 11.1 Worker Responsibilities

The worker should:

```txt
consume pending jobs
mark jobs as running
execute processing commands
persist output files
persist metadata
persist logs
mark jobs as completed or failed
update asset status
```

## 11.2 Initial Job Handlers

```txt
ExtractMetadataJobHandler
GenerateThumbnailJobHandler
TranscodeHLSJobHandler
```

## 11.3 Job Execution Flow

```txt
Job created
  ↓
Worker picks job
  ↓
Job status becomes running
  ↓
Worker writes logs
  ↓
Worker executes command
  ↓
Worker stores output
  ↓
Worker updates job output
  ↓
Job status becomes completed
```

Failure flow:

```txt
Job created
  ↓
Worker picks job
  ↓
Job status becomes running
  ↓
Command fails
  ↓
Worker captures error
  ↓
Worker writes error logs
  ↓
Job status becomes failed
  ↓
Asset status may become failed
```

---

# 12. FFmpeg / ffprobe Commands

## 12.1 Metadata Extraction

Example:

```bash
ffprobe \
  -v quiet \
  -print_format json \
  -show_format \
  -show_streams \
  input.mp4
```

## 12.2 Thumbnail Generation

Example:

```bash
ffmpeg \
  -ss 00:00:03 \
  -i input.mp4 \
  -frames:v 1 \
  -q:v 2 \
  thumbnail.jpg
```

## 12.3 HLS Encoding

Initial simple version:

```bash
ffmpeg \
  -i input.mp4 \
  -codec:v libx264 \
  -codec:a aac \
  -hls_time 6 \
  -hls_playlist_type vod \
  -hls_segment_filename "segment_%03d.ts" \
  index.m3u8
```

Future adaptive version:

```bash
ffmpeg \
  -i input.mp4 \
  -filter_complex "[0:v]split=3[v1][v2][v3];[v1]scale=w=1920:h=1080[v1out];[v2]scale=w=1280:h=720[v2out];[v3]scale=w=854:h=480[v3out]" \
  -map "[v1out]" -map 0:a -c:v:0 libx264 -b:v:0 5000k -c:a:0 aac -b:a:0 128k \
  -map "[v2out]" -map 0:a -c:v:1 libx264 -b:v:1 2800k -c:a:1 aac -b:a:1 128k \
  -map "[v3out]" -map 0:a -c:v:2 libx264 -b:v:2 1400k -c:a:2 aac -b:a:2 128k \
  -f hls \
  -hls_time 6 \
  -hls_playlist_type vod \
  -master_pl_name master.m3u8 \
  -var_stream_map "v:0,a:0 v:1,a:1 v:2,a:2" \
  stream_%v.m3u8
```

---

# 13. Storage Requirements

## 13.1 Local Storage

Use MinIO for local development.

Initial bucket:

```txt
ultramedia
```

Suggested path structure:

```txt
assets/{assetId}/source/input.mp4
assets/{assetId}/thumbnails/thumbnail.jpg
assets/{assetId}/hls/master.m3u8
assets/{assetId}/hls/segment_000.ts
assets/{assetId}/hls/segment_001.ts
```

## 13.2 Future Storage

The storage layer should be compatible with:

```txt
AWS S3
Cloudflare R2
Backblaze B2
Hetzner Object Storage
MinIO
```

---

# 14. Database Requirements

Recommended database:

```txt
PostgreSQL
```

Initial tables:

```txt
assets
media_files
processing_jobs
job_logs
```

## 14.1 Assets Table

```sql
CREATE TABLE assets (
  id UUID PRIMARY KEY,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  source_file_id UUID,
  duration DOUBLE PRECISION,
  width INTEGER,
  height INTEGER,
  bitrate INTEGER,
  video_codec TEXT,
  audio_codec TEXT,
  frame_rate DOUBLE PRECISION,
  container_format TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## 14.2 Media Files Table

```sql
CREATE TABLE media_files (
  id UUID PRIMARY KEY,
  asset_id UUID NOT NULL REFERENCES assets(id),
  kind TEXT NOT NULL,
  bucket TEXT NOT NULL,
  path TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## 14.3 Processing Jobs Table

```sql
CREATE TABLE processing_jobs (
  id UUID PRIMARY KEY,
  asset_id UUID NOT NULL REFERENCES assets(id),
  type TEXT NOT NULL,
  status TEXT NOT NULL,
  input JSONB NOT NULL DEFAULT '{}',
  output JSONB,
  error TEXT,
  progress INTEGER,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  started_at TIMESTAMP,
  finished_at TIMESTAMP
);
```

## 14.4 Job Logs Table

```sql
CREATE TABLE job_logs (
  id UUID PRIMARY KEY,
  job_id UUID NOT NULL REFERENCES processing_jobs(id),
  level TEXT NOT NULL,
  message TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

---

# 15. Monorepo Structure

Initial structure:

```txt
ultramedia-platform/
  apps/
    dashboard/
    api/
    worker/

  packages/
    media-types/
    job-contracts/
    ffmpeg-runner/
    storage/
    config/
    logger/

  services/
    ai-worker/

  infra/
    docker/
    compose/
    opentofu/

  .ai/
    skills/
      create-feature-prd.md
      create-processing-job.md
      debug-ffmpeg-error.md
      write-adr.md
      generate-tests.md

  docs/
    adr/
    prd/
    roadmap/
```

---

# 16. Recommended Stack

```txt
Frontend: SvelteKit
API: Go
Main Worker: Go
Future AI Worker: Python
Database: PostgreSQL
Queue: Redis
Storage: MinIO / S3-compatible
Media Processing: FFmpeg + ffprobe
Local Environment: Docker Compose
Observability Future: OpenTelemetry + Prometheus + Grafana
```

---

# 17. AI-First Development Strategy

AI should be used in two different ways.

## 17.1 AI Inside the Product

Future product features:

```txt
automatic subtitles
video summary
metadata enrichment
scene detection
object detection
clip suggestions
incident explanation
encoding error analysis
SEO title generation
thumbnail suggestion
```

## 17.2 AI For Development Workflows

The repository should include AI skills to reinforce engineering workflows.

Suggested folder:

```txt
.ai/
  skills/
    create-feature-prd.md
    create-processing-job.md
    create-worker.md
    write-adr.md
    review-ffmpeg-command.md
    debug-encoding-error.md
    generate-tests.md
    create-docs.md
```

Example skill purpose:

```txt
Given a new video processing feature, generate:
- PRD
- Job contract
- Database migration
- Worker interface
- API endpoint
- Dashboard screen
- Tests
- Documentation
```

---

# 18. Milestone 1 — Local Video Processing

## 18.1 Goal

Run the platform locally and process a video from upload to HLS preview.

## 18.2 Task List

```txt
1. Create monorepo
2. Create Docker Compose with PostgreSQL, Redis, and MinIO
3. Create Go API
4. Create initial database schema
5. Create upload endpoint
6. Store original file in MinIO
7. Create Asset record
8. Create metadata extraction job
9. Run ffprobe in worker
10. Persist metadata
11. Create thumbnail generation job
12. Run FFmpeg thumbnail command
13. Persist thumbnail artifact
14. Create HLS encoding job
15. Generate HLS manifest and segments
16. Persist HLS artifacts
17. Create SvelteKit dashboard
18. List assets
19. Show asset details
20. Show job status
21. Show job logs
22. Preview HLS video
```

---

# 19. Roadmap

## Phase 1 — Processing Core

```txt
asset management
upload
storage
ffprobe metadata
thumbnail generation
HLS encoding
job queue
logs
dashboard
preview
```

## Phase 2 — AI Metadata

```txt
audio extraction
transcription
subtitles
summary
tags
chapters
cuepoints
```

## Phase 3 — AI Clipping

```txt
highlight detection
scene detection
vertical crop
subject tracking
clip export
caption generation
```

## Phase 4 — Observability / NOC

```txt
job health
queue health
encoding failures
AI error explanation
pipeline timeline
asset diagnostics
playback metrics
```

## Phase 5 — Live

```txt
RTMP ingest
HLS live output
record live to VOD
restreaming
live clipping
live monitoring
```

---

# 20. Architectural Decisions

> These four decisions now live in `docs/adr/` as 0004–0007, next to the newer ones. The numbers below are kept for reference only.

## ADR 001 — Start With Modular Monolith

Decision:

Start with API and workers separated, but avoid independent microservices in the first version.

Reason:

```txt
less operational complexity
faster MVP development
easier debugging
simpler local development
clearer internal contracts
```

---

## ADR 002 — Every Processing Step Is a Job

Decision:

Every heavy video operation should be modeled as a `ProcessingJob`.

Reason:

```txt
enables queue processing
enables retries
enables logs
enables progress tracking
enables observability
allows future migration to microservices
```

---

## ADR 003 — Storage Must Be S3-Compatible

Decision:

Use MinIO locally and keep the storage layer compatible with S3.

Reason:

```txt
simple local development
easy cloud migration
compatible with many providers
avoids vendor lock-in
```

---

## ADR 004 — AI Runs in a Separate Execution Layer

Decision:

AI features should run in dedicated workers, probably using Python.

Reason:

```txt
better AI ecosystem
keeps API simple
allows independent scaling
supports local models and external APIs
```

---

# 21. Success Metrics

MVP 1 is successful when:

```txt
A user can run the project locally.
A user can upload an MP4 file.
The system extracts metadata.
The system generates a thumbnail.
The system transcodes the video to HLS.
The dashboard shows asset status and job logs.
The generated HLS video can be previewed.
```

Technical success:

```txt
jobs are persisted
logs are persisted
artifacts are stored
failures are visible
the system can be extended with new job types
```

---

# 22. Open Questions

Questions to answer during implementation:

```txt
Should the API create all jobs at once or should workers chain jobs?
Should HLS output start with single rendition or adaptive ladder?
Should files be served directly from MinIO or through the API?
Should the dashboard use polling or WebSockets for job status?
Should Redis be used as a real queue or only as a first simple implementation?
Should Go workers call FFmpeg directly or use a wrapper package?
What is the minimum metadata model needed for future AI workflows?
```

---

# 23. Next Document

After this PRD, the next document should be:

```txt
Technical Roadmap — Milestone 1: Video Processing Core
```

It should define:

```txt
repository setup
Docker Compose
database migrations
API contracts
worker contracts
queue flow
FFmpeg command wrappers
dashboard screens
first GitHub issues
```
