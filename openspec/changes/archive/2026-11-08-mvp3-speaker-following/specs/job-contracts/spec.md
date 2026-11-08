# Spec Delta

## ADDED Requirements

### Requirement: Jobs refeitos juntos
O pacote SHALL definir `RerunWith`, com os jobs que rodam de novo depois de um job refeito à mão porque usam o resultado dele: `suggest-clips` leva `track-faces`.

#### Scenario: Cortes refeitos
- **WHEN** `RerunWith["suggest-clips"]` é lido
- **THEN** SHALL ser `["track-faces"]`
