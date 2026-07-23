# Technical Roadmap — Milestone 1: Video Processing Core

**Project:** UltraMedia Platform  
**Milestone:** 1  
**Scope:** Video Processing Core  
**Status:** Draft  
**Version:** 0.1  
**Owner:** Rodrigo França  

---

# 1. Milestone Goal

Build the first working version of UltraMedia Platform where a user can upload an MP4 video, process it with FFmpeg/ffprobe, store the generated artifacts, inspect jobs/logs, and preview the generated HLS output in a dashboard.

The full flow should be:

```txt
Upload MP4
  ↓
Create Asset
  ↓
Store Source File
  ↓
Create Processing Jobs
  ↓
Extract Metadata
  ↓
Generate Thumbnail
  ↓
Transcode to HLS
  ↓
Store Artifacts
  ↓
Show Jobs and Logs
  ↓
Preview Video
```

---

# 2. Milestone Outcome

At the end of this milestone, the platform should support:

```txt
local development environment
Go API
Go worker
SvelteKit dashboard
PostgreSQL persistence
Redis queue
MinIO object storage
FFmpeg processing
ffprobe metadata extraction
HLS output generation
basic asset dashboard
job status tracking
job logs
video preview
```

This milestone is successful when a user can run the platform locally and complete the full video processing flow without manual FFmpeg commands.

---

# 3. Recommended Duration

Suggested duration:

```txt
3 to 5 weeks
```

Suggested rhythm:

```txt
Sprint 0 — Repository and local environment
Sprint 1 — API, database, and storage
Sprint 2 — Worker and processing jobs
Sprint 3 — Dashboard and preview
Sprint 4 — Hardening, docs, and demo
```

---

# 4. Milestone Strategy

The milestone should focus on one thing:

> Build the smallest complete video pipeline.

Avoid expanding scope during this milestone.

Do not add yet:

```txt
AI transcription
automatic subtitles
adaptive bitrate ladder
live streaming
restreaming
AI NOC
object detection
clip generation
multi-tenant auth
billing
DRM
ad insertion
```

The correct target is not “advanced video platform” yet.

The correct target is:

> A working video processing loop with strong architecture.

---

# 5. High-Level Architecture

```txt
SvelteKit Dashboard
        ↓
Go API
        ↓
PostgreSQL
        ↓
Redis Queue
        ↓
Go Worker
        ↓
FFmpeg / ffprobe
        ↓
MinIO Storage
```

## 5.1 Components

| Component | Responsibility |
|---|---|
| `dashboard` | User interface for upload, assets, jobs, logs, and preview |
| `api` | HTTP API, upload handling, asset creation, job creation |
| `worker` | Consumes jobs and executes FFmpeg/ffprobe commands |
| `postgres` | Stores assets, files, jobs, and logs |
| `redis` | Queue backend for processing jobs |
| `minio` | S3-compatible object storage for source and generated files |
| `ffmpeg` | Media processing |
| `ffprobe` | Technical metadata extraction |

---

# 6. Monorepo Structure

Initial target structure:

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

  infra/
    compose/
      docker-compose.yml

  docs/
    prd/
      ultramedia-platform-prd-v0.1.md
    roadmap/
      milestone-1-video-processing-core.md
    adr/
      001-modular-monolith.md
      002-processing-jobs.md
      003-s3-compatible-storage.md

  .ai/
    skills/
      create-processing-job.md
      debug-ffmpeg-error.md
      write-adr.md
      generate-tests.md

  README.md
```

---

# 7. Epics

## Epic 1 — Repository Foundation

Set up the monorepo and development environment.

### Deliverables

```txt
monorepo structure
basic README
Docker Compose
environment variables
development scripts
shared config package
```

### Tasks

- Create repository.
- Create folder structure.
- Add root `README.md`.
- Add `.env.example`.
- Add `docker-compose.yml`.
- Add PostgreSQL service.
- Add Redis service.
- Add MinIO service.
- Add API service placeholder.
- Add worker service placeholder.
- Add dashboard placeholder.
- Add development commands.

### Acceptance Criteria

- Developer can clone the repo.
- Developer can run local infrastructure.
- PostgreSQL, Redis, and MinIO start successfully.
- README explains how to start the project.

---

## Epic 2 — Database Schema

Create the initial persistence model.

### Deliverables

```txt
assets table
media_files table
processing_jobs table
job_logs table
database migration strategy
```

### Initial Tables

```txt
assets
media_files
processing_jobs
job_logs
```

### Tasks

- Choose migration tool.
- Create initial migration.
- Create `assets` table.
- Create `media_files` table.
- Create `processing_jobs` table.
- Create `job_logs` table.
- Add indexes for common queries.
- Add seed or test data if useful.

### Acceptance Criteria

- Migrations run locally.
- All initial tables are created.
- API can connect to PostgreSQL.
- Worker can connect to PostgreSQL.

---

## Epic 3 — Storage Layer

Implement S3-compatible storage using MinIO locally.

### Deliverables

```txt
storage abstraction
MinIO bucket creation
source file upload
artifact path convention
file access strategy
```

### Path Convention

```txt
assets/{assetId}/source/input.mp4
assets/{assetId}/thumbnails/thumbnail.jpg
assets/{assetId}/hls/index.m3u8
assets/{assetId}/hls/segment_000.ts
```

### Tasks

- Configure MinIO in Docker Compose.
- Create `ultramedia` bucket.
- Implement storage client.
- Implement file upload to storage.
- Implement file download or temporary local access for workers.
- Implement file URL generation.
- Document storage path conventions.

### Acceptance Criteria

- API can upload a file to MinIO.
- Worker can read source files from MinIO.
- Worker can write generated files to MinIO.
- Dashboard can access thumbnail and playback URLs.

---

## Epic 4 — API Core

Create the Go API with basic asset and job endpoints.

### Deliverables

```txt
Go API application
health endpoint
asset upload endpoint
asset list endpoint
asset detail endpoint
job list endpoint
job logs endpoint
playback endpoint
```

### Initial Endpoints

```txt
GET  /health
GET  /assets
GET  /assets/:id
POST /assets/upload
GET  /assets/:id/files
GET  /assets/:id/jobs
GET  /assets/:id/playback
GET  /jobs
GET  /jobs/:id
GET  /jobs/:id/logs
```

### Tasks

- Create Go API application.
- Add configuration loading.
- Add database connection.
- Add storage client.
- Add Redis client.
- Add upload endpoint.
- Save uploaded MP4 to MinIO.
- Create `Asset` record.
- Create `MediaFile` record for source file.
- Create initial processing jobs.
- Add asset list endpoint.
- Add asset detail endpoint.
- Add jobs endpoint.
- Add logs endpoint.
- Add playback endpoint.

### Acceptance Criteria

- API starts locally.
- Health endpoint works.
- User can upload MP4 through API.
- Asset is created in database.
- Source file is stored in MinIO.
- Processing jobs are created after upload.
- API returns asset details with files and jobs.

---

## Epic 5 — Queue and Job Orchestration

Create the basic job queue flow.

### Deliverables

```txt
job producer
job consumer
job status lifecycle
job retry foundation
job logs
```

### Initial Job Types

```txt
extract-metadata
generate-thumbnail
transcode-hls
```

### Job Statuses

```txt
pending
running
completed
failed
```

### Tasks

- Define job contract.
- Implement job creation in API.
- Push job IDs to Redis queue.
- Implement worker queue consumer.
- Load job data from PostgreSQL.
- Mark job as `running`.
- Mark job as `completed`.
- Mark job as `failed` on error.
- Persist job logs.
- Update asset status based on job results.

### Acceptance Criteria

- Jobs are created after upload.
- Worker can consume jobs.
- Job status changes are persisted.
- Logs are persisted.
- Failed jobs store error message.
- Asset status reflects processing result.

---

## Epic 6 — FFmpeg Runner

Create reusable wrappers for FFmpeg and ffprobe.

### Deliverables

```txt
ffprobe metadata extraction
thumbnail generation
HLS encoding
command execution logs
error handling
```

### Tasks

- Create `ffmpeg-runner` package.
- Implement command execution helper.
- Capture stdout and stderr.
- Implement `ffprobe` JSON metadata extraction.
- Implement thumbnail generation command.
- Implement HLS encoding command.
- Normalize command errors.
- Write logs for each processing step.

### Acceptance Criteria

- Worker can run `ffprobe`.
- Worker can parse metadata.
- Worker can generate thumbnail.
- Worker can generate HLS output.
- FFmpeg errors are visible in job logs.

---

## Epic 7 — Metadata Extraction Job

Implement the first real processing job.

### Input

```json
{
  "sourceFileId": "uuid"
}
```

### Output

```json
{
  "duration": 120.5,
  "width": 1920,
  "height": 1080,
  "bitrate": 4500000,
  "videoCodec": "h264",
  "audioCodec": "aac",
  "frameRate": 29.97,
  "containerFormat": "mov,mp4,m4a,3gp,3g2,mj2"
}
```

### Tasks

- Download source file from storage.
- Run `ffprobe`.
- Parse metadata.
- Update `assets` table.
- Write job logs.
- Mark job as completed or failed.

### Acceptance Criteria

- Metadata is extracted from uploaded MP4.
- Asset record is updated.
- Dashboard/API can show metadata.
- Job logs include command result.

---

## Epic 8 — Thumbnail Generation Job

Generate thumbnail for uploaded video.

### Input

```json
{
  "sourceFileId": "uuid",
  "timestamp": "00:00:03"
}
```

### Output

```json
{
  "thumbnailFileId": "uuid",
  "path": "assets/{assetId}/thumbnails/thumbnail.jpg"
}
```

### Tasks

- Download source file from storage.
- Run FFmpeg thumbnail command.
- Upload generated thumbnail to MinIO.
- Create `MediaFile` record.
- Write job logs.
- Mark job as completed or failed.

### Acceptance Criteria

- Thumbnail is generated.
- Thumbnail file is stored.
- Thumbnail is linked to asset.
- Asset list can show thumbnail.

---

## Epic 9 — HLS Encoding Job

Transcode MP4 to HLS output.

### Input

```json
{
  "sourceFileId": "uuid"
}
```

### Output

```json
{
  "manifestFileId": "uuid",
  "manifestPath": "assets/{assetId}/hls/index.m3u8"
}
```

### Initial FFmpeg Command

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

### Tasks

- Download source file from storage.
- Create temporary working directory.
- Run HLS encoding command.
- Upload HLS manifest to MinIO.
- Upload HLS segments to MinIO.
- Create `MediaFile` record for manifest.
- Optionally create records for segments later.
- Write job logs.
- Mark job as completed or failed.
- Update asset status to `ready` if all required jobs succeed.

### Acceptance Criteria

- HLS manifest is generated.
- HLS segments are generated.
- HLS files are stored.
- Playback endpoint returns manifest URL.
- Dashboard can preview HLS video.

---

## Epic 10 — Dashboard Foundation

Create the SvelteKit dashboard.

### Deliverables

```txt
asset list page
upload page
asset detail page
job list component
job logs component
video preview component
```

### Pages

```txt
/assets
/assets/:id
/upload
/jobs
```

### Tasks

- Create SvelteKit app.
- Add base layout.
- Add API client.
- Create upload page.
- Create asset list page.
- Create asset detail page.
- Create job list component.
- Create job logs component.
- Create video preview component.
- Add polling for asset/job status.
- Add error states.
- Add loading states.

### Acceptance Criteria

- User can upload video from UI.
- Asset appears in asset list.
- Asset detail shows metadata.
- Asset detail shows jobs.
- Asset detail shows logs.
- Asset detail previews HLS when ready.

---

## Epic 11 — Playback Preview

Enable HLS preview in the dashboard.

### Deliverables

```txt
playback endpoint
HLS manifest URL
browser playback integration
fallback states
```

### Tasks

- Implement `GET /assets/:id/playback`.
- Return HLS manifest URL.
- Configure MinIO CORS if needed.
- Add HLS playback support in dashboard.
- Use native HLS where supported.
- Use `hls.js` or UltraMedia where needed.
- Show processing state when HLS is not ready.
- Show error state when playback URL is unavailable.

### Acceptance Criteria

- Ready asset can be played in dashboard.
- Processing asset shows current state.
- Failed asset shows useful error state.
- Playback works locally.

---

## Epic 12 — Documentation and Demo

Document the milestone and prepare a demo flow.

### Deliverables

```txt
README
local setup guide
architecture overview
API documentation
worker documentation
demo script
known limitations
```

### Tasks

- Update README.
- Document Docker Compose setup.
- Document environment variables.
- Document upload flow.
- Document job flow.
- Document storage path convention.
- Document FFmpeg requirements.
- Add demo script.
- Add known limitations.
- Add next steps.

### Acceptance Criteria

- New developer can run the project locally.
- New developer can process a test video.
- Documentation explains architecture clearly.
- Demo flow is reproducible.

---

# 8. Suggested Implementation Order

## Step 1 — Repository Setup

```txt
Create monorepo
Create README
Create apps/api
Create apps/worker
Create apps/dashboard
Create packages folder
Create docs folder
```

## Step 2 — Local Infrastructure

```txt
Add Docker Compose
Add PostgreSQL
Add Redis
Add MinIO
Add .env.example
Validate services
```

## Step 3 — Database

```txt
Add migration tool
Create initial migration
Create database models
Validate DB connection from API
Validate DB connection from worker
```

## Step 4 — Storage

```txt
Configure MinIO
Create bucket
Implement storage client
Upload test file
Read test file
Generate file URL
```

## Step 5 — API Upload Flow

```txt
Create POST /assets/upload
Validate MP4 file
Store file in MinIO
Create Asset
Create MediaFile
Create ProcessingJobs
Push jobs to Redis
```

## Step 6 — Worker Base

```txt
Create worker process
Connect to Redis
Consume job IDs
Load job from PostgreSQL
Update job status
Write logs
Handle failure
```

## Step 7 — Metadata Job

```txt
Download source file
Run ffprobe
Parse JSON
Update Asset metadata
Complete job
```

## Step 8 — Thumbnail Job

```txt
Download source file
Run FFmpeg thumbnail command
Upload thumbnail
Create MediaFile
Complete job
```

## Step 9 — HLS Job

```txt
Download source file
Run FFmpeg HLS command
Upload manifest
Upload segments
Create manifest MediaFile
Complete job
Update Asset to ready
```

## Step 10 — Dashboard

```txt
Create upload page
Create asset list
Create asset detail
Show metadata
Show jobs
Show logs
Add polling
```

## Step 11 — Playback

```txt
Create playback endpoint
Return manifest URL
Configure CORS
Add HLS player
Preview generated HLS
```

## Step 12 — Hardening

```txt
Improve errors
Improve logs
Clean temp files
Add basic tests
Update docs
Record demo flow
```

---

# 9. Suggested GitHub Issues

## Foundation

```txt
[Foundation] Create monorepo structure
[Foundation] Add Docker Compose with PostgreSQL, Redis and MinIO
[Foundation] Add environment configuration
[Foundation] Add initial README
```

## Database

```txt
[Database] Add migration tooling
[Database] Create assets table
[Database] Create media_files table
[Database] Create processing_jobs table
[Database] Create job_logs table
```

## API

```txt
[API] Create Go API application
[API] Add health endpoint
[API] Add database connection
[API] Add storage client
[API] Implement POST /assets/upload
[API] Implement GET /assets
[API] Implement GET /assets/:id
[API] Implement GET /assets/:id/jobs
[API] Implement GET /jobs/:id/logs
[API] Implement GET /assets/:id/playback
```

## Worker

```txt
[Worker] Create Go worker application
[Worker] Connect worker to Redis
[Worker] Consume pending jobs
[Worker] Implement job lifecycle
[Worker] Persist job logs
[Worker] Add failure handling
```

## FFmpeg

```txt
[FFmpeg] Create command execution helper
[FFmpeg] Implement ffprobe metadata extraction
[FFmpeg] Implement thumbnail generation
[FFmpeg] Implement HLS encoding
[FFmpeg] Capture stdout and stderr logs
```

## Storage

```txt
[Storage] Configure MinIO bucket
[Storage] Implement source upload
[Storage] Implement artifact upload
[Storage] Implement file URL generation
[Storage] Document storage paths
```

## Dashboard

```txt
[Dashboard] Create SvelteKit app
[Dashboard] Create upload page
[Dashboard] Create asset list page
[Dashboard] Create asset detail page
[Dashboard] Create job status component
[Dashboard] Create job logs component
[Dashboard] Add HLS preview
[Dashboard] Add polling for job status
```

## Documentation

```txt
[Docs] Document local setup
[Docs] Document architecture
[Docs] Document API endpoints
[Docs] Document worker flow
[Docs] Document FFmpeg commands
[Docs] Create milestone demo script
```

---

# 10. Minimal Database Indexes

Suggested indexes:

```sql
CREATE INDEX idx_media_files_asset_id ON media_files(asset_id);
CREATE INDEX idx_processing_jobs_asset_id ON processing_jobs(asset_id);
CREATE INDEX idx_processing_jobs_status ON processing_jobs(status);
CREATE INDEX idx_processing_jobs_type ON processing_jobs(type);
CREATE INDEX idx_job_logs_job_id ON job_logs(job_id);
```

---

# 11. Minimal Environment Variables

```env
APP_ENV=development

API_PORT=8080

DATABASE_URL=postgres://ultramedia:ultramedia@localhost:5432/ultramedia?sslmode=disable

REDIS_URL=redis://localhost:6379

S3_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY=ultramedia
S3_SECRET_KEY=ultramedia
S3_BUCKET=ultramedia
S3_REGION=us-east-1
S3_FORCE_PATH_STYLE=true

PUBLIC_API_URL=http://localhost:8080
PUBLIC_STORAGE_URL=http://localhost:9000/ultramedia
```

---

# 12. Initial Docker Compose Services

```txt
postgres
redis
minio
api
worker
dashboard
```

In the very beginning, `api`, `worker`, and `dashboard` may run outside Docker for faster development.

A good first approach:

```txt
Docker Compose runs infrastructure only:
- PostgreSQL
- Redis
- MinIO

Local processes run:
- API
- Worker
- Dashboard
```

This gives faster feedback while building.

---

# 13. Job Creation Strategy

For MVP 1, use a simple strategy:

```txt
After upload:
  create extract-metadata job
  create generate-thumbnail job
  create transcode-hls job
  push all jobs to queue
```

This means the jobs may run independently.

However, there is one important rule:

```txt
Asset should only become ready after all required jobs are completed.
```

Future improvement:

```txt
Use workflow orchestration:
  extract metadata
    ↓
  generate thumbnail
    ↓
  transcode HLS
    ↓
  mark ready
```

For MVP 1, independent jobs are acceptable if the asset readiness logic is correct.

---

# 14. Asset Status Strategy

Initial asset statuses:

```txt
created
processing
ready
failed
```

Suggested rules:

```txt
created:
  asset exists but no job started yet

processing:
  at least one required job is pending or running

ready:
  all required jobs completed successfully

failed:
  at least one required job failed
```

Required jobs for MVP 1:

```txt
extract-metadata
generate-thumbnail
transcode-hls
```

---

# 15. File Strategy

## Source File

```txt
kind: source
path: assets/{assetId}/source/input.mp4
```

## Thumbnail File

```txt
kind: thumbnail
path: assets/{assetId}/thumbnails/thumbnail.jpg
```

## HLS Manifest

```txt
kind: manifest
path: assets/{assetId}/hls/index.m3u8
```

## HLS Segments

For MVP 1, segment records are optional.

Recommended first approach:

```txt
Store segment files in MinIO.
Do not create one database record per segment yet.
Only create MediaFile record for the manifest.
```

Reason:

```txt
simpler database
less noise
manifest references segments
enough for playback preview
```

---

# 16. Testing Strategy

## MVP Testing Priorities

Start with practical integration tests.

### API Tests

```txt
health endpoint
upload endpoint
asset list endpoint
asset detail endpoint
playback endpoint
```

### Worker Tests

```txt
metadata job with valid MP4
thumbnail job with valid MP4
HLS job with valid MP4
job failure with invalid input
```

### Storage Tests

```txt
upload file
download file
generate URL
upload generated artifact
```

### Manual E2E Test

```txt
start infra
start API
start worker
start dashboard
upload MP4
wait for processing
open asset detail
inspect metadata
inspect logs
play HLS
```

---

# 17. Demo Script

Use this script as the first public demo flow:

```txt
1. Start local environment.
2. Open dashboard.
3. Upload a short MP4.
4. Show asset created as processing.
5. Show processing jobs:
   - extract-metadata
   - generate-thumbnail
   - transcode-hls
6. Open job logs.
7. Show extracted metadata.
8. Show generated thumbnail.
9. Show HLS playback preview.
10. Explain next step: AI metadata pipeline.
```

This demo is strong because it proves the whole foundation.

---

# 18. Definition of Done

Milestone 1 is complete when:

```txt
Repository structure is created.
Local infra runs with Docker Compose.
API can upload MP4 files.
Source files are stored in MinIO.
Assets are persisted in PostgreSQL.
Processing jobs are created.
Worker consumes jobs.
ffprobe extracts metadata.
FFmpeg generates thumbnail.
FFmpeg generates HLS output.
Generated files are stored in MinIO.
Job statuses are visible.
Job logs are visible.
Dashboard lists assets.
Dashboard shows asset detail.
Dashboard previews HLS video.
Documentation explains how to run the project.
```

---

# 19. Known Risks

## Risk 1 — FFmpeg Complexity

Even simple encoding can fail with different input files.

Mitigation:

```txt
Start with short MP4 files.
Support only common H.264/AAC inputs first.
Capture logs well.
Document known limitations.
```

## Risk 2 — Storage URLs and CORS

HLS playback may fail if manifest and segments are not accessible correctly.

Mitigation:

```txt
Configure MinIO CORS early.
Keep manifest and segments in the same folder.
Validate URLs in browser.
Test playback before polishing UI.
```

## Risk 3 — Scope Creep

The project can easily expand too fast.

Mitigation:

```txt
Do not add AI in Milestone 1.
Do not add live in Milestone 1.
Do not add adaptive ladder in Milestone 1.
Do not add multi-tenant auth in Milestone 1.
```

## Risk 4 — Queue Overengineering

There is a temptation to build a complex orchestrator too soon.

Mitigation:

```txt
Use a simple Redis queue.
Persist source of truth in PostgreSQL.
Keep job lifecycle simple.
```

---

# 20. Recommended First Pull Requests

## PR 1 — Repository Bootstrap

Includes:

```txt
folder structure
README
.env.example
basic docs folders
```

## PR 2 — Local Infrastructure

Includes:

```txt
docker-compose.yml
PostgreSQL
Redis
MinIO
setup instructions
```

## PR 3 — Database Schema

Includes:

```txt
migration tool
initial tables
indexes
database connection test
```

## PR 4 — API Upload

Includes:

```txt
Go API
POST /assets/upload
storage upload
asset creation
source media file creation
job creation
```

## PR 5 — Worker Base

Includes:

```txt
Go worker
Redis consumer
job lifecycle
job logs
failure handling
```

## PR 6 — Metadata Job

Includes:

```txt
ffprobe runner
metadata parsing
asset metadata update
job completion
```

## PR 7 — Thumbnail Job

Includes:

```txt
FFmpeg thumbnail command
artifact upload
media file record
dashboard thumbnail support
```

## PR 8 — HLS Job

Includes:

```txt
FFmpeg HLS command
manifest and segment upload
playback endpoint
asset ready status
```

## PR 9 — Dashboard MVP

Includes:

```txt
upload page
asset list
asset detail
jobs
logs
preview
```

## PR 10 — Docs and Demo

Includes:

```txt
setup guide
architecture overview
demo script
known limitations
```

---

# 21. Next Milestone Preview

After Milestone 1, the natural next milestone is:

```txt
Milestone 2 — AI Metadata Pipeline
```

Expected scope:

```txt
extract audio
transcribe audio
generate subtitles
generate summary
generate tags
generate chapters
show transcript in dashboard
preview subtitles in player
```

Milestone 1 must stay focused because Milestone 2 depends on a stable asset, job, storage, and dashboard foundation.
