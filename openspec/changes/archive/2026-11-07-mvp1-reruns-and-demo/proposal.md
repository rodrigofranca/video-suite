# Proposal

## Why

O pipeline do Milestone 1 funcionava, mas faltavam três itens da Definition of Done e do hardening do roadmap: não havia como rodar de novo um job que falhou (nem gerar metadados e miniatura para vídeos enviados antes da etapa 3), rodar o projeto exigia Go, Node e FFmpeg instalados, e não havia roteiro de demo nem lista de limitações.

## What Changes

- `POST /assets/{id}/jobs` deixa de ser stub: roda um job de novo como uma nova tentativa, só quando nenhum job do asset está pendente ou rodando (ADR 0015).
- Dashboard: botão **Run again** na última tentativa de cada job e marcação das tentativas.
- Imagens Docker da API, do worker (com FFmpeg) e do dashboard (build estático com nginx).
- Profile `app` no Compose com migração automática; `make demo`, `make demo-logs` e `make sample`.
- O dashboard passa a usar `@sveltejs/adapter-static` (SPA com fallback para `index.html`).
- `docs/demo.md` e a seção "Known limitations" no README.

## Capabilities

### New Capabilities

- `job-rerun`: rodar de novo um job de um asset.
- `demo-stack`: subir a plataforma inteira em Docker com um comando.

### Modified Capabilities

- `api-skeleton`: o último stub (disparo manual de job) sai.
- `dashboard`: botão para rodar um job de novo e histórico de tentativas.
- `monorepo-structure`: novos targets do Makefile; `make down` também para a pilha de demo.

## Impact

- Código: `apps/api` (rota de re-run), `packages/media-core/store` (`CreateJobIfIdle`), `apps/dashboard`.
- Infra: `infra/docker/` (Dockerfiles e nginx), `.dockerignore`, serviços `migrate`, `api`, `worker` e `dashboard` no `docker-compose.yml`.
- Dependências: `@sveltejs/adapter-static` 3.0.10 no lugar de `adapter-auto`; imagens `golang:1.25.14-alpine3.23`, `alpine:3.23.8`, `node:22.23.4-alpine`, `nginx:1.30.6-alpine`, `migrate/migrate:v4.19.1`.
