# Demo script

A 5-minute walkthrough of the pipeline, from upload to AI summary. It needs Docker and, for the summary, an OpenCode API key.

## Before the demo

```sh
cp .env.example .env   # set OPENCODE_API_KEY for summaries, tags and chapters
make demo              # builds and starts everything (first build takes a few minutes)
make llm-check         # should print OK: the language model answers through the gateway
make sample            # creates samples/sample.mp4: 30 s, 720p, with a tone (no speech)
```

For the transcription and summary part, use an MP4 of a minute or more with someone speaking about a few topics (a phone recording works; chapters only appear for videos that change topic), or `make sample-speech` if your FFmpeg has flite (Ubuntu and Debian builds do). The first `make demo` downloads the Whisper model, so give the AI worker a minute before the first upload.

Open http://localhost:5173. Keep `make demo-logs` running in a terminal if you want to show the worker output.

## Script

1. **The empty dashboard.** "UltraMedia turns raw video into playable, inspectable assets. Everything runs locally: Go API, Go worker with FFmpeg, PostgreSQL, Redis Streams and S3-compatible storage."
2. **Upload.** Click **Upload**, pick `samples/sample.mp4`. "The file streams straight to object storage; nothing is buffered in the API."
3. **The pipeline.** The video page opens while it processes. "Every step is a job with its own status and logs: metadata with ffprobe, a thumbnail, then HLS encoding." Point at the encoding percentage and the thumbnail behind it.
4. **Logs.** Click **HLS encoding** to open its logs: the exact FFmpeg command, timings, segments uploaded. "When something fails, this is where you see why."
5. **Metadata.** Show the details card (duration, resolution, codecs, bitrate) and open **ffprobe JSON**.
6. **Playback.** Press play. "This is HLS, read straight from storage, played with hls.js."
7. **Transcript and subtitles.** "After the video is playable, the Go worker extracts the audio and a Python worker transcribes it with Whisper, locally." Turn on the subtitles in the player, then click a line of the transcript to jump there.
8. **Summary, tags and chapters.** "Then a language model reads the transcript and writes a summary, tags and chapters. The call goes through a LiteLLM gateway, so the model or provider can change without touching the code." Click a chapter to jump there.
9. **Clips.** "The same model picks the moments that work as standalone clips." Click **Preview** on the first clip, then **9:16 blur**: seconds later the vertical clip plays on the page, with captions that highlight each word as it is said, and can be downloaded.
10. **Run a job again.** Open **Thumbnail**, click **Run again**. "A new attempt is added; the earlier one stays in the history."
11. **The list.** Go back to **Videos**: thumbnail, duration, resolution, status.
12. **What's next.** "Next: vertical clips that follow the speaker's face, and editing a clip's start and end."

## After the demo

```sh
make down     # stops everything; data is kept in Docker volumes
```

To start from an empty state, also remove the volumes: `docker compose --profile app down -v`.

See the [known limitations](../README.md#known-limitations) before showing it to others.
