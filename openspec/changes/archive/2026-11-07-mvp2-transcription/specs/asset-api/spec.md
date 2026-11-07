# Spec Delta

## MODIFIED Requirements

### Requirement: Detalhe do asset
`GET /assets/{id}` SHALL retornar o asset com `source`, `jobs` (na ordem do pipeline), `artifacts`, `playback` (que é `null` enquanto o asset não está `ready`), `thumbnailUrl`, `metadata` e `subtitles` (lista com `language` e `url`, a mais recente por idioma).

#### Scenario: Asset em processamento
- **WHEN** o asset está `processing`
- **THEN** a resposta SHALL trazer os jobs com status e `playback: null`

#### Scenario: ID inexistente ou malformado
- **WHEN** o ID não existe ou não é um UUID
- **THEN** SHALL responder 404 com `{"error": "not found"}`

#### Scenario: Antes dos metadados
- **WHEN** o job de metadata ainda não terminou
- **THEN** `thumbnailUrl` e `metadata` SHALL ser `null`

#### Scenario: Antes da transcrição
- **WHEN** a transcrição ainda não terminou
- **THEN** `subtitles` SHALL ser uma lista vazia

## ADDED Requirements

### Requirement: Transcrição
`GET /assets/{id}/transcript` SHALL retornar `language`, `model`, `text`, `segments` (com `start`, `end` e `text`) e `createdAt`.

#### Scenario: Ainda não transcrito
- **WHEN** não há transcrição para o asset
- **THEN** SHALL responder 404 com `{"error": "transcript not available yet"}`
