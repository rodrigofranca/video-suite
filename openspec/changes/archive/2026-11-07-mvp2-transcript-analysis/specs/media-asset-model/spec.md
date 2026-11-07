# Spec Delta

## ADDED Requirements

### Requirement: Análises
A migração `0010` SHALL criar a tabela `analyses` (`asset_id`, `job_id`, `language`, `summary`, `tags` como `TEXT[]`, `chapters` em JSONB, `model`, `created_at`) com no máximo um registro por asset e remoção em cascata com o asset.

#### Scenario: Análise substituída
- **WHEN** uma segunda análise é salva para o mesmo asset
- **THEN** SHALL existir um único registro, com os dados da segunda
