# UltraMedia Platform

An open source, modular video operations platform. It turns raw video files into structured, playable and observable media assets: upload, metadata extraction, thumbnails, HLS encoding, transcription and subtitles, AI summaries, tags and chapters, AI-suggested clips exported for social media, job logs and playback preview.

> **Status:** Milestone 1 (video processing core) is complete. Milestone 2 (AI metadata) is under way: speech transcription and subtitles (local Whisper), and summaries, tags and chapters written by a language model through a LiteLLM gateway, are done. Phase 3 (AI clipping) has started: the model suggests clips and the worker exports them as MP4, horizontal or vertical (following the face of whoever is speaking), with burned-in captions in five styles: highlighted word, word box, karaoke, one word at a time, or lines, in the spoken language or translated by the model into English, Spanish, Portuguese, French, German or Italian, and with an opening and closing vignette and a watermark from a preset. See the [roadmap](docs/ultramedia-platform-roadmap-milestone-1.md) and the [known limitations](#known-limitations).

## Architecture

```txt
SvelteKit dashboard ──► Go API ──► PostgreSQL (assets, jobs, logs, transcripts, analyses)
       │                  │
       │                  ├──► Redis stream "jobs" ────► Go worker ──────► FFmpeg
       │                  │    Redis stream "jobs:ai" ─► Python AI worker ─► Whisper
       │                  │                                   │  └────────► LiteLLM gateway ──► OpenCode
       │                  └──────────► S3 storage ◄───────────┘  (sources, HLS, audio, subtitles)
       │                                   │
       └── HLS, thumbnails, subtitles ◄────┘  only hls/, thumbnails/ and subtitles/ are public
```

An upload creates an asset and eight jobs: `extract-metadata` → `generate-thumbnail` → `transcode-hls` → `extract-audio` → `transcribe-audio` → `analyze-transcript` → `suggest-clips` → `track-faces`. The API queues the first; each worker runs its jobs, records status, progress and logs in PostgreSQL, and queues the next on the right stream. The asset is `ready` (playable) when the HLS job succeeds; the transcript, subtitles, summary, suggested clips and who to keep in frame in each clip, following whoever speaks ([ADR 0022](docs/adr/0022-face-following-vertical-clips.md), [ADR 0023](docs/adr/0023-follow-the-active-speaker.md)) arrive a few seconds later. Exporting a clip (`export-clip`) is a job too, run on request ([ADR 0020](docs/adr/0020-ai-clips.md)); captions in another language first run `translate-transcript` on the AI worker, once per language ([ADR 0029](docs/adr/0029-translated-captions.md)). Language model calls go through the LiteLLM gateway, the only place that knows the provider and its key ([ADR 0018](docs/adr/0018-llm-gateway-litellm.md)). Metadata, thumbnail and audio jobs read the source through signed URLs (only the bytes they need).

| Path | What it is |
|---|---|
| `apps/api` | HTTP API (Go) and database migrations |
| `apps/worker` | Media worker (Go): reads the `jobs` stream, runs FFmpeg, chains the pipeline |
| `apps/ai-worker` | AI worker (Python, faster-whisper, OpenCV, ONNX Runtime): reads the `jobs:ai` stream, transcribes speech, asks the gateway for summaries, tags, chapters and clips, and finds who to keep in frame in each clip (the face, or whoever speaks) |
| `apps/dashboard` | Web dashboard (SvelteKit): upload, video list, job logs, player |
| `packages/job-contracts` | Job types, typed inputs, pipeline order, stream routing and queue message format (shared with Python through `testdata/contracts.json`) |
| `packages/media-core` | Domain entities, SQL (`store`), S3 client (`storage`), DB pool |
| `infra/` | Local infrastructure config (bucket CORS and public-read policy, LiteLLM models) |
| `docs/` | PRD, roadmap and architecture decisions ([map](docs/README.md)) |
| `openspec/` | Specs and change proposals |

## Quick start (Docker only)

```sh
cp .env.example .env   # then set OPENCODE_API_KEY (see below)
make demo              # builds and starts everything; first build takes a few minutes
make sample            # optional: creates samples/sample.mp4 to upload
```

Open http://localhost:5173 and upload an MP4; videos with speech also get a transcript, subtitles, a summary, tags and chapters. The first start downloads the Whisper model (about 140 MB) into a Docker volume. `make demo-logs` follows the API and worker logs; `make down` stops everything. A walkthrough for presenting it is in [docs/demo.md](docs/demo.md).

**Summaries need a language model.** In the [OpenCode](https://opencode.ai) console, add credits to Zen (pay per use), create an API key and set `OPENCODE_API_KEY` in `.env`. Then `make llm-check` should print `OK`. The default model is Claude Haiku 5.5: a 2-minute video cost US$ 0.0005 in testing, so a 10-minute one stays under US$ 0.001. OpenCode Go and the free models do not work here (Go is meant for coding agents; the free models only work inside OpenCode). Without a key everything else works and only the jobs that use the model (**Summary**, **Clips** and caption **Translation**) fail, saying which key to check. To use another model or provider, edit [`infra/litellm/config.yaml`](infra/litellm/config.yaml) and restart the gateway (`docker compose restart litellm`).

## Development

### Requirements

- Go 1.25+
- Python 3.12+ and [uv](https://docs.astral.sh/uv/) (AI worker)
- Node.js 22+ and npm
- Docker with Compose v2
- FFmpeg with libx264, including `ffprobe` (the worker refuses to start without them)

### Running from source

```sh
cp .env.example .env
make up        # PostgreSQL, Redis, S3 storage (RustFS) and the LiteLLM gateway; creates the bucket
make migrate   # applies database migrations
```

Then, in separate terminals:

```sh
make dev-api        # http://localhost:8080/healthz
make dev-worker
make dev-ai-worker   # downloads the Whisper model (about 140 MB) on first start
make dev-dashboard  # http://localhost:5173
```

Open http://localhost:5173, click **Upload**, pick an MP4 and watch the jobs run: metadata and thumbnail take a moment, then HLS encoding shows its progress. When it finishes, the video plays on the same page. The storage console is at http://localhost:9001 (user `ultramedia`, password `ultramedia`).

## API

| Method | Path | Returns |
|---|---|---|
| `POST` | `/assets/upload` | Upload (multipart field `file`, `.mp4`, up to 2 GB) → `201 {"assetId", "status"}` |
| `GET` | `/assets?status=` | Assets, newest first, with `thumbnailUrl` and `metadata` summary |
| `GET` | `/assets/{id}` | Asset with source, jobs, artifacts, `thumbnailUrl`, `metadata` and `playback` (when ready) |
| `DELETE` | `/assets/{id}` | Delete the video, everything recorded about it and all its stored files → `204`; `409` while uploading or while any job is pending or running |
| `GET` | `/assets/{id}/metadata` | Metadata summary and full ffprobe output |
| `GET` | `/assets/{id}/jobs` | Jobs in pipeline order, with `progress` (0–100) |
| `GET` | `/assets/{id}/artifacts` | Files produced by jobs |
| `GET` | `/assets/{id}/playback` | `{"type": "hls", "manifestUrl"}`, or `409` while not ready |
| `GET` | `/assets/{id}/transcript` | Language, model, full text and timed segments; `404` until transcribed |
| `GET` | `/assets/{id}/analysis` | Summary, tags, chapters (`start` in seconds, `title`), language and model; `404` until analyzed |
| `GET` | `/assets/{id}/clips` | Suggested clips, best first, each with the state of its exports per format; `wordTimings` (caption styles that follow each word are possible); `transcriptLanguage`; and `translations`, the languages the transcript has been translated into |
| `DELETE` | `/assets/{id}/clips/{clipId}` | Delete a clip and its exported files → `204` (the export jobs stay in the history); `409` while any job of the asset is pending or running |
| `PATCH` | `/assets/{id}/clips/{clipId}` | Adjust a clip before exporting it: any of `{"start": 12.5, "end": 40, "title": "..."}` (seconds from the start of the video; 3 s to 3 min, inside the video). New times clear the face track and queue `track-faces` for that clip; `409` while any job of the asset is pending or running |
| `POST` | `/assets/{id}/clips/{clipId}/exports` | Export a clip: `{"format": "original" \| "vertical" \| "vertical-face" \| "vertical-blur", "captions": true, "captionStyle": "highlight" \| "box" \| "karaoke" \| "one-word" \| "lines", "captionPosition": "bottom" \| "middle" \| "top", "captionColor": "yellow" \| "green" \| "pink" \| "blue" \| "violet", "captionCaps": true, "captionLanguage": "en" \| "es" \| "pt" \| "fr" \| "de" \| "it", "presetId": "<id>"}` (all but the format optional: each style has its own color and case; no language means the spoken one; a preset brings its opening and closing and the caption choices left out) → `202` with the export job, which waits for a translation when the transcript has none in that language yet; `409` while any job of the asset is pending or running |
| `GET` | `/clips/{id}/download?format=` | `302` to a 15-minute signed link that saves the exported file (`&inline=1` plays it instead) |
| `GET` | `/usage?days=30` | Language model usage of the period (`days`: 1, 7, 30, 90, or 0 for everything): calls, tokens in and out, and cost in US dollars, in total and per day, model, action and video ([ADR 0032](docs/adr/0032-ai-usage.md)); each video's own is `aiUsage` in `GET /assets/{id}` |
| `POST` | `/vignettes/upload` | Add an opening or closing to the library: multipart `name` (optional, before the file) and `file` (MP4, up to 100 MB; up to 30 s, checked on export) → `201` ([ADR 0030](docs/adr/0030-vignettes-and-presets.md)) |
| `GET` | `/vignettes` | The library, newest first, each with `usedBy` (the presets that use it) |
| `GET` | `/vignettes/{id}/file` | `302` to a signed link that plays the vignette |
| `DELETE` | `/vignettes/{id}` | `204`; `409` while a preset uses it |
| `POST` | `/watermarks/upload` | Add a watermark image: multipart `name` (optional, before the file) and `file` (PNG or JPEG, up to 5 MB and 4096 px a side) → `201` ([ADR 0031](docs/adr/0031-watermark.md)) |
| `GET` | `/watermarks` | The images, newest first, each with its size and `usedBy` |
| `GET` | `/watermarks/{id}/file` | `302` to a signed link that shows the image |
| `DELETE` | `/watermarks/{id}` | `204`; `409` while a preset uses it |
| `GET` / `POST` | `/presets` | List presets by name / create one: `{"name", "captions", "captionStyle", "captionPosition", "captionColor", "captionCaps", "captionLanguage", "introHorizontalId", "introVerticalId", "outroHorizontalId", "outroVerticalId", "watermarkId", "watermarkPosition": "top-right" \| "top-left" \| "bottom-right" \| "bottom-left", "watermarkSize": 5–40, "watermarkOpacity": 10–100}` (all but the name optional; the size is a percentage of the clip's shorter side) → `201`; `409` when the name is taken |
| `GET` / `PUT` / `DELETE` | `/presets/{id}` | One preset / replace its choices / delete it (`204`) |
| `POST` | `/assets/{id}/jobs` | Run one job again: `{"type": "generate-thumbnail"}` → `202` with the new job; `409` while any job of the asset is pending or running |
| `GET` | `/jobs/{id}` | One job |
| `GET` | `/jobs/{id}/logs` | Job logs |
| `GET` | `/healthz` | Database check |

## Common tasks

| Command | What it does |
|---|---|
| `make help` | List all targets |
| `make test` | Go and Python tests (`make test-go`, `make test-ai`). Tests that need PostgreSQL, Redis, storage or FFmpeg are skipped when those are missing |
| `make build` | Build all Go modules |
| `make migrate-down` | Revert all migrations |
| `make demo` / `make demo-logs` | Run everything in Docker / follow its API and worker logs |
| `make sample` | Create `samples/sample.mp4` (tone, no speech) with the worker image's FFmpeg |
| `make sample-speech` | Create `samples/speech.mp4` with synthetic English speech (needs a local FFmpeg with flite, as on Ubuntu/Debian) |
| `make llm-check` | Ask the language model one short question through the gateway, to check `OPENCODE_API_KEY` |
| `make down` | Stop all containers, including the demo stack (data volumes are kept) |

## Configuration

All settings come from environment variables; see [`.env.example`](.env.example).

| Variable | Used by | Default |
|---|---|---|
| `DATABASE_URL` | API, worker, migrations | required |
| `REDIS_URL` | API, worker | required |
| `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET` | API, worker | required |
| `S3_REGION` | API, worker | `us-east-1` |
| `S3_PUBLIC_URL` | API (playback URLs and clip download links: the address browsers reach) | `S3_ENDPOINT` |
| `API_PORT` | API | `8080` |
| `DASHBOARD_ORIGIN` | API (CORS) | `http://localhost:5173` |
| `PUBLIC_API_URL` | dashboard | `http://localhost:8080` |
| `WORKER_NAME` | worker | hostname. Keep it stable so a restarted worker resumes unfinished jobs |
| `WORKER_TEMP_DIR` | workers (work files) | system temp dir |
| `WHISPER_MODEL` | AI worker | `base` (`tiny`, `small`, `medium`... bigger is more accurate and slower) |
| `WHISPER_DEVICE`, `WHISPER_COMPUTE_TYPE` | AI worker | `cpu`, `int8` |
| `AI_WORKER_NAME` | AI worker | hostname; keep it stable, like `WORKER_NAME` |
| `HF_HOME` | AI worker (model cache) | `~/.cache/huggingface`; `/models` volume in Docker |
| `LLM_BASE_URL` | AI worker | empty (summary jobs fail); `http://localhost:4000/v1` in `.env.example` |
| `LLM_API_KEY` | AI worker (key for the gateway) | empty |
| `LLM_MODEL` | AI worker (a model name defined in the gateway) | `ultramedia-text` |
| `LLM_TIMEOUT_SECONDS`, `LLM_MAX_INPUT_CHARS` | AI worker | `120`, `200000` (about 4 hours of speech) |
| `LITELLM_MASTER_KEY` | gateway (its admin key) | `sk-ultramedia-dev` |
| `OPENCODE_API_KEY` | gateway (provider key) | empty |
| `OPENCODE_API_BASE` | gateway | `https://opencode.ai/inference/openai/v1` (Zen) |

## Known limitations

Milestone 1 is a working core, not a production service:

- **No authentication.** Anyone who can reach the API can upload. HLS output, thumbnails and subtitles are public to anyone with the URL; the demo uses fixed development credentials.
- **H.264 in MPEG-TS only.** HLS comes in up to four qualities (1080p, 720p, 480p, 360p; [ADR 0028](docs/adr/0028-adaptive-hls.md)), all H.264/AAC in MPEG-TS segments: no HEVC/AV1 or fMP4. Videos encoded before keep their single quality until HLS runs again.
- **MP4 only, up to 2 GB.** Files must start with an MP4 `ftyp` box; MP4s that begin with another box are rejected.
- **One job at a time.** Each worker runs one job at a time, so a long encode or transcription delays the next upload. Jobs have a 30-minute limit.
- **Transcription runs on the CPU.** The `base` model is much faster than real time on a laptop CPU but makes mistakes with names and technical terms; larger models are more accurate and slower. GPU support is not set up.
- **One transcript language per video**, detected automatically; no transcript editing yet. Captions can be translated into six languages written in the Latin alphabet (the caption font has no Japanese, Chinese or Korean). A clip keeps one file per format, so exporting the same format in another language replaces the earlier file ([ADR 0029](docs/adr/0029-translated-captions.md)). Word-by-word styles in a translation place each word on the original speech, so the highlight follows the sentence, not each word.
- **The first AI worker start needs internet** to download the Whisper model and the face detector from Hugging Face, and the speaker model (4 MB) from its authors' GitHub repository; after that they are cached.
- **AI costs are as priced in the gateway.** Each call's cost comes from LiteLLM, which needs the model's price in [`infra/litellm/config.yaml`](infra/litellm/config.yaml); change it with the model. Calls without a price are counted but add nothing to the cost, and jobs from before calls were recorded were read back from their logs, without cost ([ADR 0032](docs/adr/0032-ai-usage.md)).
- **Summaries are sent to a provider.** The transcript goes to the gateway and from there to OpenCode; do not use it for content that must stay on your machine. Each video with speech costs one model call (two if the first answer is unusable).
- **Following the speaker can pick the wrong person** when two people talk at once or the listener moves their mouth a lot; without sound it follows the biggest face, and it falls back to the centre when a face is in less than 30% of the clip ([ADR 0023](docs/adr/0023-follow-the-active-speaker.md)). Caption styles that follow each word need word timings, which transcripts made before them do not have: run the transcription again (until then, clips get line captions). Suggesting clips again replaces the suggested clips but keeps the ones edited by hand ([ADR 0026](docs/adr/0026-keep-edited-clips-when-suggesting-again.md)); files already exported stay in storage.
- **Very long transcripts are not summarized.** Above `LLM_MAX_INPUT_CHARS` (about 4 hours of speech) the Summary job fails instead of cutting the transcript.
- **The gateway has one key locally.** It runs with its master key only; per-project keys need its database and come with the shared server setup. Re-running a transcription does not re-run its summary.
- **No automatic retries.** A failed job stays failed until someone runs it again from the dashboard or the API. A re-run runs only that job, plus face tracking after a new clip suggestion.
- **Crash recovery needs a stable worker name.** A restarted worker resumes its own unfinished job only if `WORKER_NAME` is unchanged.
- **Progress starts after the download.** The encoding percentage stays at 0 while the worker downloads the source.
- **Vignettes are joined with a hard cut.** No transition between a vignette and the clip (put it in the vignette); vignettes get no captions, face framing or watermark, and the watermark stays still in its corner ([ADR 0031](docs/adr/0031-watermark.md)); and everyone shares the same presets and library until there is sign-in ([ADR 0030](docs/adr/0030-vignettes-and-presets.md)).
- **Deleting cannot be undone.** There is no trash: deleting a video or a clip removes its files at once ([ADR 0027](docs/adr/0027-delete-videos-and-clips.md)).
- **The dashboard image is built for one API address.** `PUBLIC_API_URL` is fixed when the image is built (`http://localhost:8080` in the demo).
