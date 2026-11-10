# Proposal

## Why

Segundo dos três itens que subiram no backlog em 2026-11-10. A redação sabe a frase que quer, acha o momento com a busca no acervo e precisa do corte exato dela, mesmo sem o modelo de linguagem.

## What Changes

- `clips.origin` (`suggested` ou `quote`); `job_id` e `score` opcionais, exigidos nos sugeridos.
- `POST /assets/{id}/clips`: o corte de uma citação, com o título tirado da fala e um `track-faces` só para ele; `words` em `GET /assets/{id}/transcript`.
- A sugestão de cortes mantém os de citação, primeiro na ordem.
- No painel: **✂ Clip a quote** e **Find in the transcript** na transcrição, a barra da citação, a etiqueta **Quote** e a lista de cortes sem depender da sugestão.

## Capabilities

### New Capabilities

- `quote-clips`: cortes a partir de uma citação.

### Modified Capabilities

- `clip-suggestions`: mantém os cortes de citação.
- `dashboard`: cortes sem sugestão.

## Impact

- Código: migração 0021, `packages/media-core` (`domain/clip.go`, `domain/models.go`, `store/store.go`), `apps/api` (`app/clips.go`, `httpapi/clips.go`, `httpapi/assets.go`), `apps/ai-worker` (`clips.py`, `store.py`), `apps/dashboard` (`quote.ts`, `Transcript`, `QuoteBar`, `ClipCard`, `AssetPage`).
- Docs: ADR 0039, README, CONTEXT, docs/README, demo e backlog.
