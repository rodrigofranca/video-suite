# Spec Delta

## MODIFIED Requirements

### Requirement: Efeito no asset
Rodar de novo `extract-metadata` ou `generate-thumbnail` SHALL manter o status do asset. Rodar de novo `transcode-hls` SHALL pôr o asset em `processing` até o job terminar. A nova tentativa SHALL rodar sozinha, sem enfileirar os passos seguintes, exceto os de `RerunWith`, criados `pending` na mesma transação e enfileirados pelo worker depois dela.

#### Scenario: Nova conversão HLS
- **WHEN** o asset está `ready` e o HLS roda de novo
- **THEN** o asset SHALL ficar `processing` e voltar a `ready` quando o job terminar

#### Scenario: Cortes sugeridos de novo
- **WHEN** `suggest-clips` roda de novo
- **THEN** só ele SHALL ser enfileirado, e SHALL existir um `track-faces` `pending` com a entrada do asset
