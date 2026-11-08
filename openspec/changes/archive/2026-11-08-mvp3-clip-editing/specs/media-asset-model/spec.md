# Spec Delta

## ADDED Requirements

### Requirement: Cortes editados
A migração `0014` SHALL adicionar a `clips` a coluna `edited_at` (TIMESTAMPTZ, nula para cortes como foram sugeridos).

#### Scenario: Cortes antigos
- **WHEN** a migração roda num banco com cortes
- **THEN** eles SHALL ficar com `edited_at` nulo
