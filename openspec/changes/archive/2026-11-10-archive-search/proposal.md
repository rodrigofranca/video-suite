# Proposal

## Why

Pedido em 2026-11-10: subir no backlog o corte a partir de uma citação, o corte de silêncios e a busca no acervo, começando pela busca e pela citação, que se encaixam. A busca vem primeiro: ela acha o momento de onde o corte vai partir.

## What Changes

- A tabela `transcript_segments`, uma linha por trecho com índice de texto completo, mantida por um gatilho em `transcripts`; uma configuração de busca por idioma, sem acentos.
- `GET /search?q=&offset=` com os vídeos que casam e os seus melhores momentos, com as palavras marcadas.
- No painel: **Search** no menu, a caixa na lista de vídeos, a página de resultados e o vídeo aberto no momento (`?t=`), com a transcrição rolada até ele e o caminho de volta.
- O backlog com os três itens no topo.

## Capabilities

### New Capabilities

- `archive-search`: busca pelo que foi dito em todos os vídeos.

### Modified Capabilities

- `dashboard`: busca e abertura do vídeo num momento.

## Impact

- Código: migração 0020, `packages/media-core/store/search.go`, `apps/api` (`httpapi/search.go`, rota), `apps/dashboard` (`/search`, `search.ts`, `HlsPlayer`, `Transcript`, páginas de vídeos e do vídeo).
- Docs: ADR 0038, README, CONTEXT, docs/README, demo e backlog.
