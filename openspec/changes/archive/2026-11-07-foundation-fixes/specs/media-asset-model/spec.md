# Spec Delta

## MODIFIED Requirements

### Requirement: Structs Go mapeando entidades do banco
O módulo compartilhado `packages/media-core` SHALL conter structs Go para cada entidade do banco no pacote `domain`, com campos correspondentes às colunas, importáveis pela API e pelo worker.

#### Scenario: Struct Asset compilável
- **WHEN** `go build ./...` é executado em `packages/media-core`
- **THEN** a struct `Asset` SHALL compilar sem erros com campos: `ID`, `Status`, `Filename`, `CreatedAt`, `UpdatedAt`

#### Scenario: Domínio importável fora da API
- **WHEN** a API ou o worker importam `ultramedia/media-core/domain`
- **THEN** o import SHALL resolver via `go.work`, sem `replace` no `go.mod` e sem violar a regra de pacotes `internal` do Go

#### Scenario: Status disponíveis como constantes
- **WHEN** o código precisa de um status de asset ou de job
- **THEN** SHALL usar as constantes de `domain` (`AssetUploading`, `AssetProcessing`, `AssetReady`, `AssetFailed`, `JobPending`, `JobRunning`, `JobDone`, `JobFailed`), que correspondem às CHECK constraints do banco

## ADDED Requirements

### Requirement: Índices para consultas frequentes
O schema SHALL ter índices nas chaves estrangeiras usadas em consultas (`asset_id`, `job_id`) e nas colunas de status de assets e jobs, criados por uma migração própria.

#### Scenario: Índices criados pela migração
- **WHEN** `make migrate` é executado em banco vazio
- **THEN** SHALL existir índices para `media_assets(status)`, `media_sources(asset_id)`, `processing_jobs(asset_id)`, `processing_jobs(status)`, `job_logs(job_id)`, `artifacts(asset_id)`, `artifacts(job_id)` e `technical_metadata(asset_id)`

#### Scenario: Migração de índices é reversível
- **WHEN** `make migrate-down` é executado
- **THEN** os índices SHALL ser removidos junto com as tabelas, sem erro
