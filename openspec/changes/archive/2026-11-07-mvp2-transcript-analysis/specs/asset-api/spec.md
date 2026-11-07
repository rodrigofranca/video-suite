# Spec Delta

## ADDED Requirements

### Requirement: Análise da transcrição
`GET /assets/{id}/analysis` SHALL retornar `language`, `model`, `summary`, `tags` (lista), `chapters` (lista com `start` em segundos e `title`) e `createdAt`. Sem tags ou capítulos, os campos SHALL ser listas vazias, nunca `null`.

#### Scenario: Ainda não analisado
- **WHEN** não há análise para o asset
- **THEN** SHALL responder 404 com `{"error": "analysis not available yet"}`

#### Scenario: Análise refeita
- **WHEN** o job de análise roda de novo
- **THEN** a rota SHALL retornar só a análise mais recente
