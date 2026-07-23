## ADDED Requirements

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
O módulo `apps/api` SHALL conter structs Go para cada entidade do banco em pacote `internal/domain`, com campos correspondentes às colunas.

#### Scenario: Struct Asset compilável
- **WHEN** `go build ./...` é executado em `apps/api`
- **THEN** a struct `Asset` SHALL compilar sem erros com campos: `ID`, `Status`, `Filename`, `CreatedAt`, `UpdatedAt`
