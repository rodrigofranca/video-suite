# Spec Delta

## ADDED Requirements

### Requirement: Edição nos cortes da API
Cada corte de `GET /assets/{id}/clips` SHALL ter `editedAt` (nulo se nunca editado), e cada exportação com arquivo SHALL ter `outdated: true` quando o arquivo foi cortado com outro início ou fim. O CORS SHALL permitir `PATCH`.

#### Scenario: Corte editado
- **WHEN** o título de um corte é mudado
- **THEN** a lista SHALL trazer o corte com `editedAt` preenchido
