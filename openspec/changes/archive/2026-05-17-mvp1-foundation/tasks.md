## 1. Monorepo e Go Workspace

- [x] 1.1 Criar estrutura de diretórios: `apps/api`, `apps/worker`, `apps/dashboard`, `packages/job-contracts`
- [x] 1.2 Inicializar `go.mod` em `apps/api`, `apps/worker` e `packages/job-contracts`
- [x] 1.3 Criar `go.work` na raiz referenciando os 3 módulos Go
- [x] 1.4 Criar `Makefile` na raiz com targets: `up`, `down`, `migrate`, `migrate-down`, `dev-api`, `dev-worker`, `dev-dashboard`, `help`
- [x] 1.5 Criar `.env.example` com variáveis: `DATABASE_URL`, `REDIS_URL`, `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`

## 2. Docker Compose

- [x] 2.1 Criar `docker-compose.yml` com serviços: `postgres` (5432), `redis` (6379), `minio` (9000/9001)
- [x] 2.2 Configurar volumes nomeados para persistência de dados em dev
- [x] 2.3 Verificar que `make up` sobe todos os serviços e `make down` remove tudo

## 3. Migrações do banco de dados

- [x] 3.1 Adicionar `golang-migrate` como dependência em `apps/api`
- [x] 3.2 Criar migração `0001_create_media_assets` com tabela `media_assets` (id UUID PK, filename, status CHECK, created_at, updated_at)
- [x] 3.3 Criar migração `0002_create_media_sources` com tabela `media_sources` (id, asset_id FK, storage_key, mime_type, size_bytes, checksum)
- [x] 3.4 Criar migração `0003_create_processing_jobs` com tabela `processing_jobs` (id, asset_id FK, type, input_json, status CHECK, progress, error_message, created_at, started_at, completed_at)
- [x] 3.5 Criar migração `0004_create_job_logs` com tabela `job_logs` (id, job_id FK, level, message, timestamp)
- [x] 3.6 Criar migração `0005_create_artifacts` com tabela `artifacts` (id, asset_id FK, job_id FK, type, storage_key, metadata_json, created_at)
- [x] 3.7 Criar migração `0006_create_technical_metadata` com tabela `technical_metadata` (id, asset_id FK, metadata_json, extracted_at)
- [x] 3.8 Verificar que `make migrate` aplica todas as migrações em banco limpo e `make migrate-down` as reverte

## 4. Pacote job-contracts

- [x] 4.1 Criar `packages/job-contracts/contracts.go` com constantes `JobTypeExtractMetadata`, `JobTypeGenerateThumbnail`, `JobTypeTranscodeHLS`
- [x] 4.2 Implementar struct `ExtractMetadataInput` com campos `AssetID string` e `StorageKey string`
- [x] 4.3 Implementar struct `GenerateThumbnailInput` com campos `AssetID string`, `StorageKey string`, `AtSecond float64`
- [x] 4.4 Implementar struct `TranscodeHLSInput` com campos `AssetID string` e `StorageKey string`
- [x] 4.5 Verificar que `go build ./...` em `packages/job-contracts` compila sem erros

## 5. Domínio e banco em apps/api

- [x] 5.1 Criar `apps/api/internal/domain/asset.go` com struct `Asset` (ID, Filename, Status, CreatedAt, UpdatedAt)
- [x] 5.2 Criar structs `Source`, `Job`, `JobLog`, `Artifact`, `TechnicalMetadata` em `internal/domain/`
- [x] 5.3 Adicionar `pgx/v5` como dependência em `apps/api`
- [x] 5.4 Criar `apps/api/internal/db/connect.go` com função `Connect(databaseURL string) (*pgx.Conn, error)`
- [x] 5.5 Verificar que `go build ./...` em `apps/api` compila sem erros

## 6. API skeleton

- [x] 6.1 Criar `apps/api/cmd/api/main.go` que lê `DATABASE_URL` do env, conecta ao banco e inicia servidor HTTP na porta 8080
- [x] 6.2 Adicionar roteador (stdlib `net/http` ou `chi`) e registrar rotas do PRD: POST `/assets/upload`, GET `/assets`, GET `/assets/:id`, GET `/assets/:id/jobs`, POST `/assets/:id/jobs`, GET `/jobs/:id`, GET `/assets/:id/artifacts`
- [x] 6.3 Implementar handlers stub: POST `/assets/upload` retorna 501; GET `/assets` retorna `{"assets": []}`; GET `/assets/:id` retorna 404
- [x] 6.4 Implementar `GET /healthz` que pinga o banco e retorna 200 `{"status":"ok"}` ou 503
- [x] 6.5 Verificar que `make dev-api` sobe a API e `curl localhost:8080/healthz` retorna 200

## 7. Worker skeleton

- [x] 7.1 Adicionar `go-redis/v9` como dependência em `apps/worker`
- [x] 7.2 Criar `apps/worker/cmd/worker/main.go` que lê `REDIS_URL` do env e conecta ao Redis
- [x] 7.3 Implementar criação de consumer group `workers` no stream `jobs` (XGROUP CREATE com MKSTREAM, ignorar BUSYGROUP)
- [x] 7.4 Implementar loop de consumo XREADGROUP com dispatcher por `type`
- [x] 7.5 Criar handlers stub para `extract-metadata`, `generate-thumbnail`, `transcode-hls` que logam `stub: <type> called for asset <id>` e retornam nil
- [x] 7.6 Implementar XACK após execução do handler (sucesso ou erro)
- [x] 7.7 Verificar que `make dev-worker` inicia sem erro e processa job stub enviado manualmente via `redis-cli XADD`

## 8. Dashboard bootstrap

- [x] 8.1 Criar projeto SvelteKit em `apps/dashboard` via `npm create svelte@latest`
- [x] 8.2 Verificar que `make dev-dashboard` inicia o servidor de dev SvelteKit na porta 5173
