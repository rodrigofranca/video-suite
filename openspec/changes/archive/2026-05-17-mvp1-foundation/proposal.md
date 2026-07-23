## Why

UltraMedia Platform tem visão e PRD definidos, mas zero código. Sem uma fundação estável — estrutura de monorepo, modelos de dados, contratos de job e infraestrutura local — nenhuma feature do MVP1 pode ser implementada de forma coerente. A fundação precisa vir antes de qualquer feature para evitar reescritas no modelo de dados.

## What Changes

- Criação do monorepo com estrutura `apps/` e `packages/` (sem Turborepo/Nx — Makefile orquestra)
- `packages/job-contracts`: structs Go tipadas para todos os tipos de job do MVP1
- `apps/api`: projeto Go com migrações de banco e entidades core (`media_assets`, `media_sources`, `processing_jobs`, `job_logs`, `artifacts`, `technical_metadata`)
- `apps/worker`: esqueleto do worker Go com loop de consumo Redis Streams (sem lógica de processamento FFmpeg)
- `apps/dashboard`: projeto SvelteKit bootstrapado sem features
- `docker-compose.yml`: PostgreSQL + Redis + MinIO prontos para dev local
- Makefile com targets: `up`, `down`, `migrate`, `dev-api`, `dev-worker`, `dev-dashboard`

## Capabilities

### New Capabilities

- `monorepo-structure`: Estrutura de diretórios, Makefile, Docker Compose e configuração base do monorepo
- `media-asset-model`: Schema de banco de dados e structs Go para Asset, Source, Job, JobLog e Artifact
- `job-contracts`: Pacote Go com structs tipadas para os tipos de job do MVP1 (`extract-metadata`, `generate-thumbnail`, `transcode-hls`)
- `worker-skeleton`: Worker Go com conexão Redis Streams, loop de consumo e despacho por tipo de job (sem FFmpeg)
- `api-skeleton`: API Go com conexão PostgreSQL, migrações e endpoints stub

### Modified Capabilities

## Impact

- Cria toda a estrutura de arquivos do projeto (projeto novo — sem código existente)
- Dependências adicionadas: Go, SvelteKit, PostgreSQL, Redis, MinIO, FFmpeg (runtime), `go-redis`, `golang-migrate`
- Sem breaking changes (projeto zerado)
