# asset-api Specification

## Purpose
Define as rotas de leitura da API que o dashboard e outras ferramentas usam para listar assets e inspecionar jobs, logs, artefatos e playback.

## Requirements

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

### Requirement: Informações de playback
`GET /assets/{id}/playback` SHALL retornar `{"type": "hls", "manifestUrl"}` para um asset `ready`, com a URL pública da playlist montada a partir de `S3_PUBLIC_URL`.

#### Scenario: Asset ainda não pronto
- **WHEN** o asset não está `ready`
- **THEN** SHALL responder 409 com o status atual

### Requirement: Jobs, logs e artefatos
A API SHALL expor `GET /assets/{id}/jobs`, `GET /assets/{id}/artifacts`, `GET /jobs/{id}` e `GET /jobs/{id}/logs`, com os logs em ordem cronológica.

#### Scenario: Logs de um job
- **WHEN** `GET /jobs/{id}/logs` é chamado para um job que já rodou
- **THEN** SHALL retornar `{"logs": [...]}` com nível, mensagem e horário de cada linha

### Requirement: CORS para o dashboard
A API SHALL responder com `Access-Control-Allow-Origin` apenas para a origem em `DASHBOARD_ORIGIN` (padrão `http://localhost:5173`) e SHALL responder preflights `OPTIONS` com 204.

#### Scenario: Origem não autorizada
- **WHEN** uma requisição chega com outra `Origin`
- **THEN** a resposta SHALL vir sem `Access-Control-Allow-Origin`

### Requirement: Metadados completos
`GET /assets/{id}/metadata` SHALL retornar o documento gravado pelo job de metadata, com `summary` e a saída completa do ffprobe.

#### Scenario: Metadados ainda não extraídos
- **WHEN** o job de metadata não terminou com sucesso
- **THEN** SHALL responder 404 com `{"error": "metadata not extracted yet"}`

### Requirement: Transcrição
`GET /assets/{id}/transcript` SHALL retornar `language`, `model`, `text`, `segments` (com `start`, `end` e `text`) e `createdAt`.

#### Scenario: Ainda não transcrito
- **WHEN** não há transcrição para o asset
- **THEN** SHALL responder 404 com `{"error": "transcript not available yet"}`
