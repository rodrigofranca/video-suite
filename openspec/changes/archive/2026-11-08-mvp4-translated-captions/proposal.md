# Proposal

## Why

Pedido em 2026-11-08: exportar o corte com a legenda em outro idioma, por exemplo inglês ou espanhol, para publicar o mesmo corte para outro público.

## What Changes

- Um job novo, `translate-transcript`, no worker de IA: o modelo de linguagem traduz as falas da transcrição em lotes, fala por fala; as palavras traduzidas são posicionadas sobre a fala original; a tradução fica em `transcript_translations` e vira uma legenda WebVTT no idioma.
- A exportação aceita `captionLanguage` (inglês, espanhol, português, francês, alemão, italiano). Sem tradução no idioma, a API cria a tradução e a exportação, que espera por ela (`jobcontracts.After`).
- O worker de mídia queima a legenda da tradução, em qualquer estilo, e registra o idioma no arquivo.
- Uma transcrição nova apaga as traduções.
- O painel ganha o seletor **Language**, mostra a tradução em andamento e o idioma do arquivo, e o player oferece a legenda traduzida.

## Capabilities

### New Capabilities

- `caption-translation`: o job de tradução e o que ele guarda.

### Modified Capabilities

- `clip-export`: legendas no idioma pedido.
- `job-contracts`: o job novo, os idiomas e `After`.
- `asset-api`: idioma na exportação, traduções nos cortes e nas legendas.
- `dashboard`: seletor de idioma.

## Impact

- Código: `packages/job-contracts`, `packages/media-core` (`Translation`, store), migração 0015, `apps/ai-worker` (`translate.py`), `apps/worker` (`jobs/clip.go`), `apps/api` (`clips.go`, `assets.go`), `apps/dashboard` (`Clips`, `JobList`, `HlsPlayer`).
- Docs: ADR 0029, README, CONTEXT, docs/README e demo.
