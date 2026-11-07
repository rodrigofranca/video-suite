# Proposal

## Why

Com a fundação rodando (change `foundation-fixes`), o próximo passo do roadmap é a primeira fatia de ponta a ponta: enviar um MP4 pelo dashboard e assisti-lo em HLS. É a demo que prova o modelo de asset, a fila, o storage e o player juntos, e destrava os jobs de metadata e thumbnail. As quatro decisões em aberto (encadeamento do pipeline, acesso ao HLS, polling e player) foram confirmadas em 2026-11-07.

## What Changes

- `POST /assets/upload`: recebe o MP4 em streaming direto para o storage (até 2 GB, sem buffer em memória ou disco), confere extensão e assinatura MP4, calcula SHA-256, cria asset, source e os três jobs, e enfileira o primeiro.
- Rotas de leitura: lista e detalhe de assets, jobs, artefatos, playback e logs de job; CORS para o dashboard.
- Worker: ciclo de vida do job no PostgreSQL (`running` → `done`/`failed`), logs de job gravados em `job_logs`, limite de 30 minutos, encadeamento do próximo job, status do asset definido pelo job HLS.
- Job `transcode-hls` real com FFmpeg (H.264 + AAC, segmentos de 6 s) e upload do resultado para `assets/{id}/hls/`; metadata e thumbnail continuam stubs.
- Cliente S3 (`packages/media-core/storage`) e camada SQL (`packages/media-core/store`) compartilhados por API e worker.
- `storage-init` aplica política de leitura pública só em `assets/*/hls/*`.
- Dashboard: lista de vídeos, upload com progresso, página do vídeo com jobs, logs e player `hls.js`; polling a cada 3 s.
- ADRs 0009–0012 com as decisões confirmadas.

## Capabilities

### New Capabilities

- `asset-upload`: envio de vídeos, validação e criação do asset com seus jobs.
- `asset-api`: leitura de assets, jobs, logs, artefatos e informações de playback pela API HTTP.
- `processing-pipeline`: ciclo de vida dos jobs, encadeamento e status do asset.
- `hls-transcoding`: conversão do vídeo de origem para HLS.
- `object-storage`: organização dos arquivos no storage S3 e o que é público.
- `dashboard`: interface web para enviar, acompanhar e assistir vídeos.

### Modified Capabilities

- `api-skeleton`: upload e leitura deixam de ser stubs; a API verifica Redis e storage ao iniciar.
- `worker-skeleton`: `transcode-hls` deixa de ser stub; o worker verifica banco, storage e FFmpeg ao iniciar; job interrompido por encerramento não é ackado.
- `monorepo-structure`: o Compose aplica a política de leitura pública do HLS.

## Impact

- Código: `apps/api`, `apps/worker`, `apps/dashboard`, `packages/job-contracts`, `packages/media-core`.
- Dependências: `aws-sdk-go-v2` (`service/s3`, `feature/s3/transfermanager` v0.5.5), `hls.js` 1.7.3.
- Infra: `infra/storage/public-hls-policy.json`; novas variáveis `S3_PUBLIC_URL`, `DASHBOARD_ORIGIN`, `PUBLIC_API_URL`, `WORKER_TEMP_DIR`.
- Runtime: o worker agora precisa de FFmpeg com libx264.
