# Proposal

## Why

Primeira entrega da fase de cortes com IA (Fase 3 do PRD), confirmada em 2026-11-07. Com a transcrição, o resumo e o gateway de modelos prontos, a plataforma pode achar os momentos de um vídeo que funcionam sozinhos e exportá-los prontos para Reels, Shorts e TikTok: o caso de uso central do produto Ai-video-editor, que será construído sobre o video-suite.

## What Changes

- Job novo no fim do pipeline: `suggest-clips`, no worker de IA. O modelo escolhe até 5 trechos de 15 a 60 s, com título, motivo e nota; o worker encaixa início e fim nos trechos da transcrição.
- Job sob demanda `export-clip`, no worker Go: corta o trecho em MP4 nos formatos original, 9:16 com corte central e 9:16 com o quadro inteiro sobre fundo desfocado, com legendas queimadas opcionais.
- Migração `0011_create_clips`; `GET /assets/{id}/clips`, `POST /assets/{id}/clips/{clipId}/exports` e `GET /clips/{id}/download`.
- Cortes exportados ficam privados em `assets/{id}/clips/`; o download é um link assinado de 15 minutos.
- Dashboard: seção **Clips** com prévia no player, exportação, progresso, vídeo exportado e download.
- Worker de IA: o pedido de JSON ao modelo e a conferência da resposta passam a ser compartilhados entre análise e cortes.
- Imagem do worker: fontes DejaVu, usadas pelas legendas queimadas.

## Capabilities

### New Capabilities

- `clip-suggestions`: os trechos sugeridos pelo modelo a partir da transcrição.
- `clip-export`: a exportação de um trecho em MP4.

### Modified Capabilities

- `job-contracts`: tipos `suggest-clips` e `export-clip`, formatos de corte.
- `processing-pipeline`: sete passos.
- `asset-upload`: o upload cria os sete jobs.
- `asset-api`: rotas dos cortes.
- `dashboard`: seção de cortes.
- `media-asset-model`: tabela `clips`.
- `object-storage`: cortes privados com download por link assinado.
- `demo-stack`: fontes na imagem do worker.

## Impact

- Código: `apps/ai-worker` (`clips.py`, `answers.py`), `apps/worker` (`captions`, `ffmpeg/clip.go`, `jobs/clip.go`), `apps/api` (`clips.go`), `apps/dashboard` (`Clips.svelte`), `packages/job-contracts`, `packages/media-core`.
- Custo: uma chamada a mais ao modelo por vídeo com fala (cerca de US$ 0,0006 num vídeo de 2 minutos com Claude Haiku 5.5).
- Docs: ADR 0020.
