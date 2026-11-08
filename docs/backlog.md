# Backlog

Ideas agreed on but not scheduled yet, newest first. When one is picked up it
becomes an OpenSpec change and leaves this list.

## Clips

- **Warn before suggesting clips again**: a new suggestion replaces the list, including clips edited by hand; the dashboard could ask first, or keep the edited ones.

## Platform

- **Shared LiteLLM instance**: point `LLM_BASE_URL`, `LLM_API_KEY` and `LLM_MODEL` at the shared gateway (with a key for UltraMedia) once it exists; the local `litellm` service stays for development.
- **Production readiness**: authentication, deleting videos and clips, and HLS in several qualities (adaptive bitrate).
