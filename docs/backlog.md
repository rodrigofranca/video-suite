# Backlog

Ideas agreed on but not scheduled yet, newest first. When one is picked up it
becomes an OpenSpec change and leaves this list. Effort is a rough guess:
small (a day or less), medium (a few days), large (a week or more).

The journalism, social media and newsroom ideas were added on 2026-11-09.
On 2026-11-10 three of them were moved to the top, in this order: search the
archive (done, docs/adr/0038), clip from a quote (done, docs/adr/0039), and
cutting silences and filler words. After them: transcript correction and
glossary; post copy; face blurring and source credits; sign-in with approval,
then publishing to the networks.

## Next

1. **Cut silences and filler words** (medium): remove pauses and fillers ("uh", "like") with jump cuts, as common in Reels and TikTok; the word timings say where the speech is.

## Journalism

- **Transcript correction and glossary** (medium): edit the transcript's words before they reach captions and translations, and keep a glossary of names, acronyms and places that Whisper and the language model are given, so a wrong name is not burned into a clip.
- **Speakers and name captions** (medium to large): tell the speakers apart, let someone name each once (name and role), and burn a name caption (lower third) when that person appears; builds on the speaker tracking (docs/adr/0023).
- **Face blurring** (medium): blur chosen faces through a clip, for anonymous sources, minors or people who did not consent; reuses the face detector (docs/adr/0022).
- **Source and date credit** (small): a credit line such as "Images: TV Câmara, 08/11" burned in, kept in the preset next to the watermark (docs/adr/0031).
- **Claims to check** (small): the language model lists numbers and factual claims with their time in the video, for the newsroom to verify; it flags, it does not judge.

## Social media

- **Post copy per network** (small): a title, caption, hashtags and an X thread for each clip, in the clip's language, written by the language model.
- **More shapes** (medium): 1:1 and 4:5 for feeds, and a split screen for interviews and debates that stacks the two people in a 9:16 clip.
- **Export with one click** (small): the preset also keeps the formats, and one button exports all of them.
- **Clip cover** (small to medium): pick a frame and put a title over it, as the cover for Shorts and Reels.
- **Subtitle files** (small): download a clip's captions as SRT or VTT, for platforms where captions are uploaded as a file.
- **Publish and schedule** (large): send clips to YouTube, Instagram, TikTok and X, now or at a set time; needs sign-in and each network's authorization.
- **Transitions for vignettes** (small): a short fade between a vignette and the clip, as a preset choice.

## Newsroom

- **Ingest from a link or live** (large): bring in a press conference, a session or a live stream from its address, and cut clips while it is happening.
- **Parallel workers and GPU transcription** (medium): run several jobs at once and transcribe on a GPU, so a busy news day does not queue up.

## Platform

- **Production readiness: sign-in, roles and approval** (large): accounts with email and password, each person seeing their own videos and presets and admins seeing everything; roles (reporter, editor) and an approval step before a clip is published.
- **Shared LiteLLM instance**: point `LLM_BASE_URL`, `LLM_API_KEY` and `LLM_MODEL` at the shared gateway (with a key for UltraMedia) once it exists; the local `litellm` service stays for development.
