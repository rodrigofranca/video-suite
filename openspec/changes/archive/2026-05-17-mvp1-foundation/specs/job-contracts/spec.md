## ADDED Requirements

### Requirement: Pacote job-contracts com structs tipadas
O módulo `packages/job-contracts` SHALL definir structs Go para os 3 tipos de job do MVP1: `ExtractMetadataInput`, `GenerateThumbnailInput`, `TranscodeHLSInput`.

#### Scenario: Pacote compila de forma isolada
- **WHEN** `go build ./...` é executado em `packages/job-contracts`
- **THEN** o pacote SHALL compilar sem dependências externas além da stdlib

#### Scenario: Structs têm campos mínimos documentados
- **WHEN** o código de `job-contracts` é inspecionado
- **THEN** `ExtractMetadataInput` SHALL ter pelo menos `AssetID` e `StorageKey`
- **THEN** `GenerateThumbnailInput` SHALL ter pelo menos `AssetID`, `StorageKey` e `AtSecond`
- **THEN** `TranscodeHLSInput` SHALL ter pelo menos `AssetID` e `StorageKey`

### Requirement: Constantes para tipos de job
O pacote SHALL exportar constantes string para os tipos de job: `JobTypeExtractMetadata`, `JobTypeGenerateThumbnail`, `JobTypeTranscodeHLS`.

#### Scenario: Constantes usáveis por API e Worker
- **WHEN** API ou Worker importam `job-contracts`
- **THEN** as constantes SHALL estar disponíveis sem cast ou string literal hardcoded

### Requirement: job-contracts importável por API e Worker via Go Workspace
API e Worker SHALL importar `job-contracts` sem `replace` directives no próprio `go.mod`, usando apenas `go.work` na raiz.

#### Scenario: Import resolve via go.work
- **WHEN** `go build ./...` é executado em `apps/api` ou `apps/worker`
- **THEN** o import de `job-contracts` SHALL resolver sem erro
