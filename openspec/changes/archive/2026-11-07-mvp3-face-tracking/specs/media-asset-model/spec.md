# Spec Delta

## ADDED Requirements

### Requirement: Rosto nos cortes
A migração `0013` SHALL adicionar a `clips` a coluna `face_track` (JSONB, nula até o rastreio), com `coverage`, `points`, `width` e `height`.

#### Scenario: Cortes antigos
- **WHEN** a migração roda num banco com cortes
- **THEN** eles SHALL ficar com `face_track` nulo
