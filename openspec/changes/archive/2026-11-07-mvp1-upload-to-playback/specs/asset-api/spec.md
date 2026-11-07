# Spec Delta

## Purpose
Define as rotas de leitura da API que o dashboard e outras ferramentas usam para listar assets e inspecionar jobs, logs, artefatos e playback.

## ADDED Requirements

### Requirement: Listagem de assets
`GET /assets` SHALL retornar `{"assets": [...]}` com os assets do mais novo para o mais antigo, aceitando o filtro opcional `status`.

#### Scenario: Filtro por status
- **WHEN** `GET /assets?status=ready` é chamado
- **THEN** SHALL retornar apenas assets `ready`

#### Scenario: Status desconhecido
- **WHEN** `GET /assets?status=weird` é chamado
- **THEN** SHALL responder 400

### Requirement: Detalhe do asset
`GET /assets/{id}` SHALL retornar o asset com `source`, `jobs` (na ordem do pipeline), `artifacts` e `playback`, que é `null` enquanto o asset não está `ready`.

#### Scenario: Asset em processamento
- **WHEN** o asset está `processing`
- **THEN** a resposta SHALL trazer os três jobs com status e `playback: null`

#### Scenario: ID inexistente ou malformado
- **WHEN** o ID não existe ou não é um UUID
- **THEN** SHALL responder 404 com `{"error": "not found"}`

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
