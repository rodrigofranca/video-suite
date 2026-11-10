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
6. **Playback.** Press play. "This is HLS, read straight from storage, played with hls.js, in up to four qualities: the player picks the one the connection can take." Open **Quality** to see **Auto** and each quality.
7. **Transcript and subtitles.** "After the video is playable, the Go worker extracts the audio and a Python worker transcribes it with Whisper, locally." Turn on the subtitles in the player, then click a line of the transcript to jump there.
8. **Summary, tags and chapters.** "Then a language model reads the transcript and writes a summary, tags and chapters. The call goes through a LiteLLM gateway, so the model or provider can change without touching the code." Click a chapter to jump there.
9. **Clips.** "The same model picks the moments that work as standalone clips." Click **Preview** on the first clip, then **9:16 face**: seconds later the vertical clip plays on the page, framed on the speaker's face (with two people on screen, it cuts to whoever is talking), with captions that highlight each word as it is said, and can be downloaded. Export **9:16 crop** too to compare it with the plain centre crop. Try another **Captions** style (word box, karaoke, one word at a time), move it to the **Top**, pick another color and **CAPS**, and export again. Then click **✎ Edit**: move the start to the next sentence, rename the clip, **Preview**, **Save**: the faces are found again for the new range, and the earlier export says it was made before the edit. Last, pick another **Language** (say English) and export **9:16 crop** again: "The model translates the transcript first, line for line, and the captions come out translated, still in time with the speech." The player now offers the translated subtitles too.
10. **Presets and vignettes.** Open **Presets**, upload a short opening (an MP4 with a logo) and a closing under **Vignettes**, and a logo under **Watermarks**, then **+ New preset**: pick a caption style and color, the opening for 9:16 clips, the closing, and the watermark (the preview shows it in both shapes). Back on the video, pick the preset on a clip and export **9:16 face**: "The preset fills in the caption choices, joins the opening and the closing and lays the logo, in one FFmpeg run; an opening made for 16:9 is shown whole over a blurred copy of itself."
11. **What the AI cost.** In the video's details, **AI usage** shows the tokens and the cost of its summary, clips and translations. Open **AI usage** in the menu: "Every call to the language model is recorded with its tokens and the cost the gateway reports; here are the totals per day, action, model and video." Hover a bar for that day's numbers.
12. **Search the archive.** Open **Search** in the menu (or type in the box on **Videos**) and search for a word said in the video, without its accents: "Every transcript is indexed by segment, with its language's rules: accents and word endings don't matter, and a phrase goes in quotes." Tap a moment: the video opens there, with the transcript scrolled to the line; **← Results** goes back.
13. **Clip from a quote.** On the video the search opened, tap **✂ Clip a quote** in the transcript, then the first and the last word of a sentence: "The clip snaps to the words, with a little room so the first syllable is not cut." **Preview**, change the title if you like, **Make the clip**: it is first in **Clips**, marked **Quote**, and exports like any other. Suggesting clips again keeps it.
14. **Run a job again.** Open **Thumbnail**, click **Run again**. "A new attempt is added; the earlier one stays in the history."
15. **The list.** Go back to **Videos**: thumbnail, duration, resolution, status.
16. **Clean up.** Open a test video, scroll to **Delete video**, confirm: "The video, its clips and every stored file are gone; the list no longer shows it."
17. **What's next.** "Next: cut silences and filler words; after that, sign-in, to run it on a server."

## After the demo

```sh
make down     # stops everything; data is kept in Docker volumes
```

To start from an empty state, also remove the volumes: `docker compose --profile app down -v`.

See the [known limitations](../README.md#known-limitations) before showing it to others.
