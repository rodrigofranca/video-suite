# Spec Delta

## ADDED Requirements

### Requirement: Transcrições
A migração `0009` SHALL criar a tabela `transcripts` (`asset_id`, `job_id`, `language`, `text`, `segments` em JSONB, `model`, `created_at`) com no máximo um registro por asset e remoção em cascata com o asset.

#### Scenario: Transcrição substituída
- **WHEN** uma segunda transcrição é salva para o mesmo asset
- **THEN** SHALL existir um único registro, com os dados da segunda
