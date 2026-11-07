# Spec Delta

## ADDED Requirements

### Requirement: Um registro de metadados por asset
A tabela `technical_metadata` SHALL ter no máximo um registro por asset, garantido por índice único em `asset_id` criado na migração `0008`.

#### Scenario: Migração em banco com duplicatas
- **WHEN** a migração `0008` roda num banco com mais de um registro para o mesmo asset
- **THEN** SHALL manter só o mais recente e criar o índice único
