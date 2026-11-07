# Proposal

## Why

Depois da change `mvp1-upload-to-playback`, os jobs de metadata e thumbnail ainda eram stubs: apareciam como concluídos sem fazer nada, e a lista de vídeos não tinha imagem, duração nem resolução. A conversão HLS também não mostrava progresso, então vídeos longos ficavam só como "running". Esta change completa as três etapas do pipeline do MVP1.

## What Changes

- `extract-metadata` real: ffprobe lê o original por URL assinada e o resultado (resumo + saída completa) fica em `technical_metadata`, um registro por asset.
- `generate-thumbnail` real: um JPEG de até 1280 px, no segundo pedido ou no meio do vídeo quando ele é mais curto, com volta ao primeiro quadro; salvo em `assets/{id}/thumbnails/` e registrado como artefato.
- `transcode-hls` reporta progresso (0–99 durante a conversão, 100 ao terminar), gravado no máximo uma vez por segundo.
- URLs assinadas nunca aparecem em logs, erros ou metadados gravados.
- Miniaturas passam a ser públicas, como o HLS.
- API: `thumbnailUrl` e `metadata` na lista e no detalhe; nova rota `GET /assets/{id}/metadata`.
- Dashboard: miniatura, duração e resolução na lista; capa do player, progresso da conversão e metadados na página do vídeo.
- Migração `0008`: um registro de `technical_metadata` por asset.
- Os stubs do worker são removidos.

## Capabilities

### New Capabilities

- `metadata-extraction`: leitura dos metadados técnicos do vídeo com ffprobe.
- `thumbnail-generation`: geração da miniatura do vídeo.

### Modified Capabilities

- `hls-transcoding`: progresso da conversão.
- `processing-pipeline`: progresso dos jobs.
- `object-storage`: miniaturas públicas e leitura do original por URL assinada.
- `asset-api`: miniatura e metadados nas respostas; rota de metadados completos.
- `dashboard`: miniaturas, metadados e progresso.
- `media-asset-model`: um registro de metadados por asset.
- `worker-skeleton`: fim dos stubs; o worker também exige `ffprobe`.
- `monorepo-structure`: a política pública do Compose inclui as miniaturas.

## Impact

- Código: `apps/worker` (handlers, ffmpeg, pipeline), `apps/api`, `apps/dashboard`, `packages/media-core` (storage, store, domain).
- Banco: migração `0008_unique_technical_metadata`.
- Infra: `infra/storage/public-read-policy.json` (antes `public-hls-policy.json`).
- Docs: ADRs 0013 e 0014.
