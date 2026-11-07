# Proposal

## Why

Segunda entrega da fase de cortes com IA, escolhida em 2026-11-07. As legendas queimadas da primeira entrega seguem as linhas do WebVTT; nos apps de vídeo curto, o padrão é mostrar poucas palavras por vez, com a palavra dita destacada. O Whisper já calcula o tempo de cada palavra, mas ele não era guardado.

## What Changes

- A transcrição guarda o tempo de cada palavra (migração `0012`, coluna `transcripts.words`).
- A exportação ganha o estilo de legenda `words` (padrão): frases curtas, com a palavra dita em amarelo; `lines` mantém o estilo anterior.
- Transcrições antigas não têm os tempos: o corte sai em `lines` e o painel avisa que basta rodar a transcrição de novo.
- API: `captionStyle` no pedido de exportação; `wordTimings` na lista de cortes; `captionStyle` em cada exportação.
- Dashboard: **Captions** vira uma escolha entre **Word by word**, **Lines** e **Off**.

## Capabilities

### Modified Capabilities

- `speech-transcription`: guarda as palavras com tempo.
- `clip-export`: estilos de legenda.
- `asset-api`: estilo no pedido e na lista.
- `dashboard`: escolha do estilo.
- `media-asset-model`: coluna `words`.

## Impact

- Código: `apps/ai-worker` (`transcribe.py`, `store.py`), `apps/worker` (`captions/words.go`, `jobs/clip.go`), `apps/api` (`clips.go`), `apps/dashboard` (`Clips.svelte`), `packages/job-contracts`, `packages/media-core`.
- Banco: cerca de 40 bytes por palavra (uns 400 KB para uma hora de fala).
- Docs: ADR 0021.
