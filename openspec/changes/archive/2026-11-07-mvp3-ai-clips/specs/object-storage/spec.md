# Spec Delta

## ADDED Requirements

### Requirement: Cortes privados
Os cortes exportados SHALL ficar em `assets/{id}/clips/{clipId}/{formato}.mp4`, sem leitura anônima, e SHALL ser entregues só por links assinados gerados pela API.

#### Scenario: Leitura anônima de um corte
- **WHEN** um navegador sem credenciais pede o arquivo de um corte direto no storage
- **THEN** o storage SHALL recusar
