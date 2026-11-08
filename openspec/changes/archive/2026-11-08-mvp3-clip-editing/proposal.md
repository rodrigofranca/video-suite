# Proposal

## Why

Pedido em 2026-11-08: o modelo acerta o momento, mas nem sempre as bordas, e o título pode ser genérico. Antes de exportar, deve dar para mudar início, fim e título de um corte, com botões para pular para a frase anterior ou a seguinte.

## What Changes

- `PATCH /assets/{id}/clips/{clipId}` com `start`, `end` e `title`, conferidos (3 s a 3 minutos, dentro do vídeo, título de até 120 caracteres).
- Novos tempos limpam o rastreio de rostos do corte e criam um `track-faces` só para ele (`clip_id` novo e opcional na entrada).
- Migração `0014`: `clips.edited_at`. A API devolve `editedAt` e marca exportações feitas antes da edição como `outdated`.
- Painel: **✎ Edit** abre um editor no card, com título, início e fim (frase anterior ou seguinte, −1 s e +1 s), **Preview** e **Save**.

## Capabilities

### New Capabilities

- `clip-editing`: editar um corte e o que acontece depois.

### Modified Capabilities

- `face-tracking`: rastrear só um corte.
- `job-contracts`: `clip_id` em `TrackFacesInput`.
- `media-asset-model`: `clips.edited_at`.
- `asset-api`: `editedAt` e `outdated`.
- `dashboard`: o editor.

## Impact

- Código: `packages/job-contracts`, `packages/media-core` (`EditClipIfIdle`), `apps/api` (rota nova, CORS com PATCH), `apps/ai-worker` (`faces.py`) e `apps/dashboard` (`ClipEditor.svelte`).
- Docs: ADR 0024, README, CONTEXT, backlog e roteiro da demo.
