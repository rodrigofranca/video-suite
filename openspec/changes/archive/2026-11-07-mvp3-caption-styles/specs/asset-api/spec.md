# Spec Delta

## MODIFIED Requirements

### Requirement: Pedir uma exportação
`POST /assets/{id}/clips/{clipId}/exports` com `{"format", "captions", "captionStyle"}` SHALL criar e enfileirar um job `export-clip` e responder 202 com o job. `captions` SHALL valer `true` e `captionStyle` SHALL valer `highlight` quando ausentes; os estilos são `highlight`, `box`, `karaoke`, `one-word` e `lines`. Formato ou estilo desconhecido SHALL responder 400; corte inexistente ou de outro asset, 404; e algum job do asset pendente ou rodando, 409.

#### Scenario: Pipeline ainda rodando
- **WHEN** a exportação é pedida enquanto o asset tem jobs pendentes
- **THEN** a API SHALL responder 409

#### Scenario: Estilo desconhecido
- **WHEN** o pedido traz `"captionStyle": "comic-sans"`
- **THEN** a API SHALL responder 400
