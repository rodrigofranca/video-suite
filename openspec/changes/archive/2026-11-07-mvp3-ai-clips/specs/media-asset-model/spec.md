# Spec Delta

## ADDED Requirements

### Requirement: Cortes sugeridos
A migração `0011` SHALL criar a tabela `clips` (`asset_id`, `job_id`, `start_seconds`, `end_seconds`, `title`, `reason`, `score` de 1 a 10, `rank`, `model`, `created_at`), com fim maior que o início e remoção em cascata com o asset.

#### Scenario: Corte invertido
- **WHEN** uma inserção tem `end_seconds` menor que `start_seconds`
- **THEN** o banco SHALL rejeitar com erro de constraint
