# Spec Delta

## MODIFIED Requirements

### Requirement: Listagem de assets
`GET /assets` SHALL retornar `{"assets": [...]}` com os assets do mais novo para o mais antigo, aceitando o filtro opcional `status`. Cada asset SHALL trazer `thumbnailUrl` e `metadata` (o resumo técnico), ambos `null` enquanto não existem.

#### Scenario: Filtro por status
- **WHEN** `GET /assets?status=ready` é chamado
- **THEN** SHALL retornar apenas assets `ready`

#### Scenario: Status desconhecido
- **WHEN** `GET /assets?status=weird` é chamado
- **THEN** SHALL responder 400

#### Scenario: Miniatura e metadados na lista
- **WHEN** um asset já tem miniatura e metadados
- **THEN** o item SHALL trazer a URL pública da miniatura e o resumo com duração e resolução

### Requirement: Detalhe do asset
`GET /assets/{id}` SHALL retornar o asset com `source`, `jobs` (na ordem do pipeline), `artifacts`, `playback` (que é `null` enquanto o asset não está `ready`), `thumbnailUrl` e `metadata`.

#### Scenario: Asset em processamento
- **WHEN** o asset está `processing`
- **THEN** a resposta SHALL trazer os três jobs com status e `playback: null`

#### Scenario: ID inexistente ou malformado
- **WHEN** o ID não existe ou não é um UUID
- **THEN** SHALL responder 404 com `{"error": "not found"}`

#### Scenario: Antes dos metadados
- **WHEN** o job de metadata ainda não terminou
- **THEN** `thumbnailUrl` e `metadata` SHALL ser `null`

## ADDED Requirements

### Requirement: Metadados completos
`GET /assets/{id}/metadata` SHALL retornar o documento gravado pelo job de metadata, com `summary` e a saída completa do ffprobe.

#### Scenario: Metadados ainda não extraídos
- **WHEN** o job de metadata não terminou com sucesso
- **THEN** SHALL responder 404 com `{"error": "metadata not extracted yet"}`
