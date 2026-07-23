## Context

Projeto novo — zero código existente. Todas as decisões de arquitetura para o MVP1 foram registradas no CONTEXT.md e ADRs (docs/adr/). A fundação deve estabelecer estrutura de monorepo, modelos de dados, contratos de job e infraestrutura local sem implementar lógica de processamento FFmpeg.

Stack: Go (API + Worker), SvelteKit (Dashboard), PostgreSQL, Redis Streams, MinIO, FFmpeg (runtime).

## Goals / Non-Goals

**Goals:**
- Monorepo com estrutura `apps/` e `packages/` funcional via Makefile
- Schema de banco completo com migrações versionadas (golang-migrate)
- Pacote `job-contracts` com structs Go para os 3 tipos de job do MVP1
- Worker com loop Redis Streams funcional (despacha por tipo, sem FFmpeg)
- API Go com endpoints stub respondendo 200/501
- Docker Compose subindo PostgreSQL + Redis + MinIO em um comando
- Developer consegue rodar `make up && make migrate` e ter infra pronta

**Non-Goals:**
- Lógica FFmpeg (metadata, thumbnail, HLS) — scope de features posteriores
- Dashboard com UI real — apenas projeto SvelteKit bootstrapado
- Autenticação/autorização
- Deploy em produção
- Turborepo/Nx — Makefile é suficiente para MVP

## Decisions

**Estrutura de monorepo sem tooling:**
Pastas `apps/api`, `apps/worker`, `apps/dashboard`, `packages/job-contracts`. Makefile orquestra builds e tasks de dev. Go Workspace (`go.work`) resolve imports entre módulos Go sem symlinks. Turborepo seria overhead antes de ter múltiplos pacotes JS.

**Go Workspace para módulos Go compartilhados:**
`job-contracts` é um módulo Go separado referenciado via `go.work`. API e Worker importam sem `replace` directives no go.mod, facilitando futuro split em repos separados.

**Redis Streams com XREADGROUP:**
Consumer group `workers` com consumer único no MVP. Job entra via XADD, worker faz XREADGROUP + XACK após processamento. Job não-ackado após crash fica em PEL (Pending Entry List) e pode ser re-claimed. Sem biblioteca de abstração de filas — go-redis direto.

**golang-migrate para migrações:**
Migrações versionadas em `apps/api/migrations/` com formato `NNNN_description.up.sql` / `.down.sql`. Makefile expõe `make migrate` e `make migrate-down`. Sem ORM — SQL puro via `pgx`.

**Schema de banco:**
6 tabelas conforme PRD: `media_assets`, `media_sources`, `processing_jobs`, `job_logs`, `artifacts`, `technical_metadata`. UUIDs como PKs (gerados pela aplicação, não pelo banco) para portabilidade futura. `status` como texto com CHECK constraint.

**MinIO como storage S3-compatible:**
Bucket único `ultramedia` no dev. SDK `aws-sdk-go-v2` com endpoint customizado — mesmo código funciona apontado para S3 em produção.

## Risks / Trade-offs

- **Go Workspace pode confundir IDEs** → Documentar setup no README; confirmar que gopls funciona com `go.work`
- **Redis Streams sem dead letter queue** → Jobs que falham ficam em PEL indefinidamente no MVP; worker deve mover para stream `jobs:failed` após N tentativas (implementar no skeleton)
- **MinIO sem TLS no dev** → Aceitável para ambiente local; produção requer configuração separada
- **Schema definido antes de implementar features** → Possível necessidade de migrações adicionais quando FFmpeg revelar requisitos não antecipados; migrações são baratas, manter schema evoluível

## Open Questions

- Versão mínima de Go? (recomendação: 1.22+ para range over integers e toolchain management)
- `pgx` diretamente ou `database/sql` + driver pgx? (recomendação: `pgx` direto — evita overhead de interface)
