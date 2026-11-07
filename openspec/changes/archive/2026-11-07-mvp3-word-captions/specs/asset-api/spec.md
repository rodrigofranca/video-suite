# Spec Delta

## MODIFIED Requirements

### Requirement: Cortes do asset
`GET /assets/{id}/clips` SHALL retornar `{"clips": [...], "wordTimings": <bool>}`, com os cortes do melhor para o pior, cada um com `id`, `start`, `end`, `duration`, `title`, `reason`, `score`, `rank`, `model` e `exports`: por formato, o job de exportação mais recente (`status`, `progress`, `error`) e, quando há arquivo, `ready`, `captions`, `captionStyle`, `width`, `height` e `sizeBytes`. `wordTimings` SHALL dizer se a transcrição tem as palavras com tempo.

#### Scenario: Exportação em andamento
- **WHEN** um corte tem uma exportação `pending` em `vertical`
- **THEN** o corte SHALL trazer essa exportação com `status: "pending"` e `ready: false`

#### Scenario: Transcrição antiga
- **WHEN** a transcrição do asset não tem palavras
- **THEN** `wordTimings` SHALL ser `false`

### Requirement: Pedir uma exportação
`POST /assets/{id}/clips/{clipId}/exports` com `{"format", "captions", "captionStyle"}` SHALL criar e enfileirar um job `export-clip` e responder 202 com o job. `captions` SHALL valer `true` e `captionStyle` SHALL valer `words` quando ausentes. Formato ou estilo desconhecido SHALL responder 400; corte inexistente ou de outro asset, 404; e algum job do asset pendente ou rodando, 409.

#### Scenario: Pipeline ainda rodando
- **WHEN** a exportação é pedida enquanto o asset tem jobs pendentes
- **THEN** a API SHALL responder 409

#### Scenario: Estilo desconhecido
- **WHEN** o pedido traz `"captionStyle": "karaoke"`
- **THEN** a API SHALL responder 400
