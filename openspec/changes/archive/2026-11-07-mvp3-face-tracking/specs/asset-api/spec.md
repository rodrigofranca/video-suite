# Spec Delta

## MODIFIED Requirements

### Requirement: Cortes do asset
`GET /assets/{id}/clips` SHALL retornar `{"clips": [...], "wordTimings": <bool>}`, do melhor corte ao pior, cada um com `id`, `start`, `end`, `duration`, `title`, `reason`, `score`, `rank`, `model`, `faceCoverage` e `exports`: por formato, a exportação mais recente (`status`, `progress`, `error`) e, com arquivo, `ready`, `captions`, `captionStyle`, `framing`, `width`, `height` e `sizeBytes`. `wordTimings` SHALL dizer se há palavras com tempo.

#### Scenario: Exportação em andamento
- **WHEN** um corte tem uma exportação `pending` em `vertical`
- **THEN** o corte SHALL trazer essa exportação com `status: "pending"` e `ready: false`

#### Scenario: Transcrição antiga
- **WHEN** a transcrição do asset não tem palavras
- **THEN** `wordTimings` SHALL ser `false`

#### Scenario: Antes e depois do rastreio de rostos
- **WHEN** o corte ainda não tem `face_track`, e depois ganha um com `coverage` 0,85
- **THEN** `faceCoverage` SHALL ser `null`, e depois `0.85`
