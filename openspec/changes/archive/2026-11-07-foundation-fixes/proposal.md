# Proposal

## Why

A change `mvp1-foundation` foi marcada como concluída, mas uma revisão em 2026-11-07 encontrou lacunas que impediam qualquer pessoa de rodar o projeto a partir do repositório:

- A pasta `apps/worker` nunca foi versionada: a regra `worker` do `.gitignore`, pensada para o binário, ignorava a pasta inteira. Como o `go.work` aponta para ela, o build da API quebrava.
- A linha `include .env 2>/dev/null || true` do Makefile não é sintaxe válida do make, então nenhum target rodava.
- A imagem `minio/minio` não existe mais no Docker Hub, então `make up` falhava.
- A API usava uma única `pgx.Conn`, que não é segura para requisições simultâneas.
- As structs de domínio e o acesso ao banco estavam em `apps/api/internal/`, que o Go não deixa o worker importar.
- As specs principais estavam no formato de delta e eram rejeitadas por `openspec validate`.

Sem corrigir isso, nenhuma feature do MVP1 (upload, jobs reais, dashboard) pode começar.

## What Changes

- `.gitignore` com regras ancoradas, que ignoram só os binários.
- `apps/worker` recriado conforme a spec `worker-skeleton`, com retomada de mensagens pendentes após queda, proteção contra panic e encerramento gracioso.
- Formato da mensagem da fila (`job_id`, `type`, `payload`) definido em `packages/job-contracts`, para API e worker usarem o mesmo código.
- Novo módulo `packages/media-core` com os pacotes `domain` (structs e constantes de status) e `db` (pool `pgxpool`).
- API usando o pool, com `API_PORT` opcional e encerramento gracioso.
- RustFS substitui o MinIO no Docker Compose; o serviço `storage-init` cria o bucket `ultramedia` e aplica CORS para o dashboard local.
- **BREAKING (ambiente local)**: variáveis `MINIO_*` renomeadas para `S3_*`; o volume `minio_data` dá lugar a `storage_data`.
- Makefile corrigido: `-include .env`, migrações via `go run` (sem instalar a CLI), `migrate-down` com `-all`, novos targets `build` e `test`.
- Migração `0007_add_lookup_indexes` com índices nas chaves estrangeiras e nos status.
- `README.md` na raiz e specs principais convertidas para o formato válido (`## Purpose` + `## Requirements`), sem mudar o conteúdo.

## Capabilities

### New Capabilities

### Modified Capabilities

- `monorepo-structure`: novo pacote `packages/media-core`, storage RustFS com bucket e CORS, Makefile sem dependência da CLI `migrate`, README.
- `media-asset-model`: structs movidas para `packages/media-core/domain`, constantes de status, índices.
- `job-contracts`: formato compartilhado da mensagem da fila.
- `worker-skeleton`: mensagens malformadas e panics são logados e ackados; retomada de mensagens pendentes; encerramento gracioso.
- `api-skeleton`: pool de conexões, porta configurável e encerramento gracioso.

## Impact

- Código: `apps/api`, `apps/worker` (novo), `packages/job-contracts`, `packages/media-core` (novo), `go.work`.
- Infra: `docker-compose.yml`, `infra/storage/cors.json`, `.env.example`, `Makefile`.
- Banco: migração `0007` (só índices, sem mudança de dados).
- Dependências: `github.com/redis/go-redis/v9` v9.22.0 (a v9.23 exige Go 1.26), `github.com/jackc/pgx/v5` v5.9.2.
- Docs: `README.md`, `docs/adr/0004`–`0008`.
