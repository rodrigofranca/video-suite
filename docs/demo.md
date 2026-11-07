# Demo script

A 5-minute walkthrough of the Milestone 1 pipeline. It needs only Docker.

## Before the demo

```sh
make demo     # builds and starts everything (first build takes a few minutes)
make sample   # creates samples/sample.mp4: 30 s, 720p, with a tone (no speech)
```

For the transcription part, use a short MP4 with someone speaking (a phone recording works), or `make sample-speech` if your FFmpeg has flite (Ubuntu and Debian builds do). The first `make demo` downloads the Whisper model, so give the AI worker a minute before the first upload.

Open http://localhost:5173. Keep `make demo-logs` running in a terminal if you want to show the worker output.

## Script

1. **The empty dashboard.** "UltraMedia turns raw video into playable, inspectable assets. Everything runs locally: Go API, Go worker with FFmpeg, PostgreSQL, Redis Streams and S3-compatible storage."
2. **Upload.** Click **Upload**, pick `samples/sample.mp4`. "The file streams straight to object storage; nothing is buffered in the API."
3. **The pipeline.** The video page opens while it processes. "Every step is a job with its own status and logs: metadata with ffprobe, a thumbnail, then HLS encoding." Point at the encoding percentage and the thumbnail behind it.
4. **Logs.** Click **HLS encoding** to open its logs: the exact FFmpeg command, timings, segments uploaded. "When something fails, this is where you see why."
5. **Metadata.** Show the details card (duration, resolution, codecs, bitrate) and open **ffprobe JSON**.
6. **Playback.** Press play. "This is HLS, read straight from storage, played with hls.js."
7. **Transcript and subtitles.** "After the video is playable, the Go worker extracts the audio and a Python worker transcribes it with Whisper, locally." Turn on the subtitles in the player, then click a line of the transcript to jump there.
8. **Run a job again.** Open **Thumbnail**, click **Run again**. "A new attempt is added; the earlier one stays in the history."
9. **The list.** Go back to **Videos**: thumbnail, duration, resolution, status.
10. **What's next.** "Next: summaries, tags and chapters from the transcript."

## After the demo

```sh
make down     # stops everything; data is kept in Docker volumes
```

To start from an empty state, also remove the volumes: `docker compose --profile app down -v`.

See the [known limitations](../README.md#known-limitations) before showing it to others.
