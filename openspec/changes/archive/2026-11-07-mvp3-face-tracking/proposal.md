# Proposal

## Why

Pedido em 2026-11-07: o 9:16 com recorte central corta quem fala quando a pessoa não está no meio do quadro, que é o caso comum em entrevistas e vídeos com duas pessoas. O corte vertical deve acompanhar o rosto.

## What Changes

- Novo job `track-faces` no fim do pipeline, no worker de IA: acha o rosto principal em cada corte sugerido (YuNet, do OpenCV) e grava um caminho de câmera por corte.
- Novo formato de exportação `vertical-face`: o recorte 9:16 segue esse caminho; sem rosto suficiente, fica no centro e diz por quê.
- API: `faceCoverage` em cada corte e `framing` em cada exportação.
- Dashboard: botão **9:16 face**, com quanto do corte tem rosto, e "centred" quando a exportação ficou no centro.

## Capabilities

### New Capabilities

- `face-tracking`: o job `track-faces` e o caminho de câmera.

### Modified Capabilities

- `job-contracts`: `track-faces` no `Pipeline` e no stream `jobs:ai`, formato `vertical-face`.
- `processing-pipeline`: `track-faces` depois de `suggest-clips`.
- `asset-upload`: oito jobs por vídeo.
- `clip-export`: formato `vertical-face`.
- `asset-api`: `faceCoverage` e `framing`.
- `dashboard`: botão **9:16 face**.
- `media-asset-model`: coluna `face_track` em `clips`.
- `ai-worker`: carrega o detector de rostos na inicialização.
- `demo-stack`: o detector fica no volume dos modelos.

## Impact

- Código: `apps/ai-worker` (`faces.py`, OpenCV e NumPy como dependências), `apps/worker` (`ffmpeg/clip.go`, `jobs/clip.go`), `packages/job-contracts`, `packages/media-core`, `apps/api` e `apps/dashboard`.
- Migração `0013_add_clip_face_track`.
- Docs: ADR 0022, README, CONTEXT, backlog e roteiro da demo.
