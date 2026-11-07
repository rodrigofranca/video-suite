# Tasks

## 1. Contratos e núcleo compartilhado

- [x] 1.1 `jobcontracts.Pipeline` e `NextJobType`
- [x] 1.2 `domain.NewID`, `IsID`, `MaxUploadBytes` e constantes de artefato e nível de log
- [x] 1.3 Cliente S3 `media-core/storage` (upload multipart, download, delete, exists, URL pública) com testes contra o RustFS
- [x] 1.4 Camada SQL `media-core/store` com testes contra o PostgreSQL
- [x] 1.5 Política de leitura pública de `assets/*/hls/*` no `storage-init` e novas variáveis no `.env.example`

## 2. API

- [x] 2.1 `POST /assets/upload` em streaming, com validação de extensão, assinatura MP4 e limite de 2 GB (com e sem `Content-Length`)
- [x] 2.2 Criação de asset, source e jobs numa transação; enfileiramento do primeiro job; falha de fila marca job e asset como `failed`
- [x] 2.3 Rotas de leitura de assets, jobs, logs, artefatos e playback; IDs malformados respondem 404
- [x] 2.4 CORS para `DASHBOARD_ORIGIN`; verificação de Redis e storage na inicialização

## 3. Worker

- [x] 3.1 Dispatcher com `jobs.Job` (ID, asset, payload, logger)
- [x] 3.2 `pipeline.Runner`: status no banco, logs em `job_logs`, timeout de 30 min, panic vira falha, entrega duplicada ignorada, interrupção não ackada
- [x] 3.3 Encadeamento do próximo job e status do asset pelo job HLS
- [x] 3.4 Executor de FFmpeg e argumentos HLS, com testes
- [x] 3.5 Handler `transcode-hls` com upload dos segmentos antes da playlist e criação do artefato
- [x] 3.6 Verificação de banco, storage e FFmpeg na inicialização

## 4. Dashboard

- [x] 4.1 Cliente da API e modo SPA
- [x] 4.2 Lista de vídeos com polling
- [x] 4.3 Upload com progresso e mensagens de erro da API
- [x] 4.4 Página do vídeo com player `hls.js`, jobs, logs e detalhes

## 5. Documentação e verificação

- [x] 5.1 ADRs 0009–0012, `CONTEXT.md`, `docs/README.md` e README
- [x] 5.2 Teste de ponta a ponta no navegador: upload, processamento, reprodução e logs
