# Spec Delta

## MODIFIED Requirements

### Requirement: Encadeamento do pipeline
Quando um job termina, com sucesso ou falha, o worker que o executou SHALL enfileirar o próximo job `pending` do mesmo asset, na ordem de `jobcontracts.Pipeline`, no stream indicado por `jobcontracts.StreamFor`. O encadeamento SHALL continuar depois do `transcode-hls`.

#### Scenario: Metadata falha e o pipeline continua
- **WHEN** `extract-metadata` falha
- **THEN** `generate-thumbnail` SHALL ser enfileirado e o asset SHALL continuar `processing`

#### Scenario: Próximo job não pode ser enfileirado
- **WHEN** o enfileiramento do próximo job falha
- **THEN** esse job SHALL ficar `failed` com a mensagem do erro e o asset SHALL ficar `failed`

#### Scenario: Depois do HLS
- **WHEN** `transcode-hls` termina
- **THEN** `extract-audio` SHALL ser enfileirado em `jobs` e, quando ele terminar, `transcribe-audio` SHALL ser enfileirado em `jobs:ai`

### Requirement: Status do asset pelo job HLS
O worker SHALL marcar o asset `ready` quando `transcode-hls` termina com sucesso e `failed` quando ele falha. Os passos seguintes SHALL não mudar o status do asset.

#### Scenario: Pipeline completo
- **WHEN** os jobs até o `transcode-hls` terminam com sucesso
- **THEN** o asset SHALL ficar `ready`, mesmo com a extração de áudio e a transcrição ainda pendentes
