# media-asset-model Specification

## Purpose
Define o modelo de dados de mídia: tabelas do PostgreSQL criadas por migrações versionadas e as structs Go que representam essas entidades.

## Requirements

### Requirement: Schema de banco de dados para entidades core
O banco de dados SHALL conter as tabelas `media_assets`, `media_sources`, `processing_jobs`, `job_logs`, `artifacts` e `technical_metadata` criadas via migrações versionadas com golang-migrate.

#### Scenario: Migrações criam todas as tabelas
- **WHEN** `make migrate` é executado em banco vazio
- **THEN** as 6 tabelas SHALL existir com todas as colunas, constraints e índices definidos

#### Scenario: Migrações são reversíveis
- **WHEN** `make migrate-down` é executado
- **THEN** as tabelas SHALL ser removidas sem erro

### Requirement: Modelo de Asset com status persistido
A tabela `media_assets` SHALL ter coluna `status` com valores controlados por CHECK constraint: `uploading`, `processing`, `ready`, `failed`.

#### Scenario: Status inválido rejeitado pelo banco
- **WHEN** uma inserção tenta definir `status = 'unknown'`
- **THEN** o banco SHALL rejeitar com erro de constraint

#### Scenario: Asset criado com status inicial correto
- **WHEN** um novo asset é inserido pela aplicação
- **THEN** `status` SHALL ser `uploading`

### Requirement: PKs como UUID gerados pela aplicação
Todas as tabelas SHALL usar UUID como chave primária. UUIDs SHALL ser gerados pela aplicação Go, não pelo banco.

#### Scenario: PK não é serial/sequence
- **WHEN** o schema é inspecionado
- **THEN** nenhuma tabela SHALL ter coluna `SERIAL`, `BIGSERIAL` ou `GENERATED ALWAYS AS IDENTITY` como PK

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

### Requirement: Índices para consultas frequentes
O schema SHALL ter índices nas chaves estrangeiras usadas em consultas (`asset_id`, `job_id`) e nas colunas de status de assets e jobs, criados por uma migração própria.

#### Scenario: Índices criados pela migração
- **WHEN** `make migrate` é executado em banco vazio
- **THEN** SHALL existir índices para `media_assets(status)`, `media_sources(asset_id)`, `processing_jobs(asset_id)`, `processing_jobs(status)`, `job_logs(job_id)`, `artifacts(asset_id)`, `artifacts(job_id)` e `technical_metadata(asset_id)`

#### Scenario: Migração de índices é reversível
- **WHEN** `make migrate-down` é executado
- **THEN** os índices SHALL ser removidos junto com as tabelas, sem erro

### Requirement: Um registro de metadados por asset
A tabela `technical_metadata` SHALL ter no máximo um registro por asset, garantido por índice único em `asset_id` criado na migração `0008`.

#### Scenario: Migração em banco com duplicatas
- **WHEN** a migração `0008` roda num banco com mais de um registro para o mesmo asset
- **THEN** SHALL manter só o mais recente e criar o índice único

### Requirement: Transcrições
A migração `0009` SHALL criar a tabela `transcripts` (`asset_id`, `job_id`, `language`, `text`, `segments` em JSONB, `model`, `created_at`) com no máximo um registro por asset e remoção em cascata com o asset.

#### Scenario: Transcrição substituída
- **WHEN** uma segunda transcrição é salva para o mesmo asset
- **THEN** SHALL existir um único registro, com os dados da segunda

### Requirement: Análises
A migração `0010` SHALL criar a tabela `analyses` (`asset_id`, `job_id`, `language`, `summary`, `tags` como `TEXT[]`, `chapters` em JSONB, `model`, `created_at`) com no máximo um registro por asset e remoção em cascata com o asset.

#### Scenario: Análise substituída
- **WHEN** uma segunda análise é salva para o mesmo asset
- **THEN** SHALL existir um único registro, com os dados da segunda

### Requirement: Cortes sugeridos
A migração `0011` SHALL criar a tabela `clips` (`asset_id`, `job_id`, `start_seconds`, `end_seconds`, `title`, `reason`, `score` de 1 a 10, `rank`, `model`, `created_at`), com fim maior que o início e remoção em cascata com o asset.

#### Scenario: Corte invertido
- **WHEN** uma inserção tem `end_seconds` menor que `start_seconds`
- **THEN** o banco SHALL rejeitar com erro de constraint

### Requirement: Palavras da transcrição
A migração `0012` SHALL adicionar a `transcripts` a coluna `words` (JSONB, `[]` por padrão), com as palavras e seus tempos.

#### Scenario: Transcrição antiga
- **WHEN** a migração roda num banco com transcrições
- **THEN** elas SHALL ficar com `words` igual a `[]`
