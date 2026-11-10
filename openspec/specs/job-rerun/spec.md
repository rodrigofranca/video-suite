# job-rerun Specification

## Purpose
Define como rodar de novo um job de um asset, por exemplo depois de uma falha ou para processar vídeos antigos, mantendo o histórico de tentativas.

## Requirements

### Requirement: Rodar um job de novo
`POST /assets/{id}/jobs` com `{"type": "<tipo>"}` SHALL criar uma nova tentativa `pending` daquele tipo, com a entrada montada a partir do source do asset, enfileirá-la e responder 202 com o job criado. Só os tipos que a tabela de tipos de job marca como "pode rodar de novo" SHALL ser aceitos.

#### Scenario: Nova tentativa de miniatura
- **WHEN** todos os jobs do asset terminaram e o pedido é `{"type": "generate-thumbnail"}`
- **THEN** SHALL existir uma nova linha de job desse tipo, enfileirada, e as tentativas anteriores SHALL continuar no histórico

#### Scenario: Tipo desconhecido ou corpo inválido
- **WHEN** o tipo não é um job do pipeline ou o corpo não é JSON
- **THEN** a API SHALL responder 400

#### Scenario: Job que roda com uma exportação
- **WHEN** o pedido é `{"type": "export-clip"}` ou `{"type": "translate-transcript"}`
- **THEN** a API SHALL responder 400 dizendo que esse job roda quando um corte é exportado

### Requirement: Só com o asset ocioso
A API SHALL recusar com 409 o pedido enquanto qualquer job do asset estiver `pending` ou `running`, ou enquanto o asset estiver `uploading`. Pedidos simultâneos SHALL resultar em no máximo uma nova tentativa.

#### Scenario: Dois pedidos seguidos
- **WHEN** dois pedidos chegam em seguida para o mesmo asset ocioso
- **THEN** o primeiro SHALL receber 202 e o segundo 409

### Requirement: Efeito no asset
Rodar de novo `extract-metadata` ou `generate-thumbnail` SHALL manter o status do asset. Rodar de novo `transcode-hls` SHALL pôr o asset em `processing` até o job terminar. A nova tentativa SHALL rodar sozinha, sem enfileirar os passos seguintes, exceto os de `RerunWith`, criados `pending` na mesma transação e postos na fila pelo relay depois dela.

#### Scenario: Nova conversão HLS
- **WHEN** o asset está `ready` e o HLS roda de novo
- **THEN** o asset SHALL ficar `processing` e voltar a `ready` quando o job terminar

#### Scenario: Cortes sugeridos de novo
- **WHEN** `suggest-clips` roda de novo
- **THEN** só ele SHALL ser enfileirado, e SHALL existir um `track-faces` `pending` com a entrada do asset
