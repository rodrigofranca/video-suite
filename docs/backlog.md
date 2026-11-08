# Backlog

Ideas agreed on but not scheduled yet, newest first. When one is picked up it
becomes an OpenSpec change and leaves this list.

## Clips

- **Watermark in presets**: a logo image over the exported clip, with its corner and size, kept in the preset next to the vignettes (docs/adr/0030).
- **Transitions for vignettes**: a short fade between a vignette and the clip, as a preset choice.

## Platform

- **Shared LiteLLM instance**: point `LLM_BASE_URL`, `LLM_API_KEY` and `LLM_MODEL` at the shared gateway (with a key for UltraMedia) once it exists; the local `litellm` service stays for development.
- **Production readiness**: authentication.
