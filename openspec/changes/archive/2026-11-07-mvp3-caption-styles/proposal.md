# Proposal

## Why

Pedido em 2026-11-07: além da palavra destacada, poder escolher entre tipos de legenda nos cortes, nos estilos mais comuns dos apps de vídeo curto. Ajustes de posição, cor e maiúsculas foram para o backlog.

## What Changes

- Cinco estilos de legenda na exportação: `highlight` (padrão, era `words`), `box`, `karaoke`, `one-word` e `lines`.
- Dashboard: **Captions** lista os cinco estilos e **Off**; cada exportação mostra o estilo usado.
- `docs/backlog.md` com os ajustes pedidos (posição, cor do destaque, maiúsculas) e os próximos passos já combinados.

## Capabilities

### Modified Capabilities

- `clip-export`: estilos de legenda.
- `asset-api`: valores de `captionStyle`.
- `dashboard`: escolha entre os estilos.

## Impact

- Código: `apps/worker/internal/captions` (`WriteHighlight`, `WriteBox`, `WriteKaraoke`, `WriteOneWord`), `jobs/clip.go`, `packages/job-contracts`, `apps/api` e `apps/dashboard`.
- Docs: ADR 0021 atualizada, backlog novo.
