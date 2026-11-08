# Proposal

## Why

Pedido em 2026-11-08: sugerir os cortes de novo apagava, sem aviso, os cortes ajustados à mão. Em vez de só avisar, a nova sugestão deve manter o que foi editado, e o painel deve dizer o que vai mudar antes de rodar.

## What Changes

- `suggest-clips` substitui só os cortes não editados; descarta sugestões novas que repetem mais da metade de um editado; ordena todos pela nota.
- Sem nada a cortar, os editados ficam.
- Painel: **Run again** do job de cortes pede confirmação, dizendo quantos cortes serão substituídos e quantos editados ficam.

## Capabilities

### Modified Capabilities

- `clip-suggestions`: a sugestão mantém os editados.
- `clip-editing`: edições sobrevivem a uma nova sugestão.
- `dashboard`: confirmação antes de sugerir de novo.

## Impact

- Código: `apps/ai-worker` (`clips.py`, `store.py`), `packages/media-core/store` (`ReplaceClips`) e `apps/dashboard` (`JobList`, página do vídeo).
- Docs: ADR 0026, README, CONTEXT; o item saiu do backlog.
