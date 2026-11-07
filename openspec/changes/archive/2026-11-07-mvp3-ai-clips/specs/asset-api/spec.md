# Spec Delta

## ADDED Requirements

### Requirement: Cortes do asset
`GET /assets/{id}/clips` SHALL retornar `{"clips": [...]}` do melhor para o pior, cada um com `id`, `start`, `end`, `duration`, `title`, `reason`, `score`, `rank`, `model` e `exports`: por formato, o job de exportação mais recente (`status`, `progress`, `error`) e, quando há arquivo, `ready`, `captions`, `width`, `height` e `sizeBytes`.

#### Scenario: Exportação em andamento
- **WHEN** um corte tem uma exportação `pending` em `vertical`
- **THEN** o corte SHALL trazer essa exportação com `status: "pending"` e `ready: false`

### Requirement: Pedir uma exportação
`POST /assets/{id}/clips/{clipId}/exports` com `{"format", "captions"}` SHALL criar e enfileirar um job `export-clip` e responder 202 com o job. `captions` SHALL valer `true` quando ausente. Formato desconhecido SHALL responder 400; corte inexistente ou de outro asset, 404; e algum job do asset pendente ou rodando, 409.

#### Scenario: Pipeline ainda rodando
- **WHEN** a exportação é pedida enquanto o asset tem jobs pendentes
- **THEN** a API SHALL responder 409

### Requirement: Download de um corte
`GET /clips/{id}/download?format=...` SHALL redirecionar (302) para um link assinado de 15 minutos do arquivo exportado, que salva o arquivo como `<video>-clip-<n>-<formato>.mp4`; com `inline=1`, o link SHALL tocar no navegador. Sem arquivo nesse formato, SHALL responder 404.

#### Scenario: Nome do arquivo
- **WHEN** o corte 1 do vídeo `My Talk (final).mp4` é baixado em `vertical`
- **THEN** o arquivo SHALL se chamar `My-Talk-final-clip-1-vertical.mp4`
