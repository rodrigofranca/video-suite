# Backlog

Ideas agreed on but not scheduled yet, newest first. When one is picked up it
becomes an OpenSpec change and leaves this list.

## Clip captions

Asked for on 2026-11-07, when the caption styles were added. Today each style
comes with a fixed position, colors and case.

- **Position**: bottom, middle or top of the picture.
- **Highlight color**: a few ready-made colors (yellow, green, pink, blue) for the highlighted word, the box and the karaoke fill.
- **Capitals**: turn all-caps on or off for every style (today only *One word at a time* uses capitals).

## Clips

- **Adjust a clip before exporting it**: change its start, end and title, with buttons to snap to the previous or next sentence.

## Platform

- **Shared LiteLLM instance**: point `LLM_BASE_URL`, `LLM_API_KEY` and `LLM_MODEL` at the shared gateway (with a key for UltraMedia) once it exists; the local `litellm` service stays for development.
- **Production readiness**: authentication, deleting videos and clips, and HLS in several qualities (adaptive bitrate).
