# Spec Delta

## MODIFIED Requirements

### Requirement: Encadeamento do pipeline
Quando um job termina, com sucesso ou falha, o worker que o executou SHALL enfileirar o próximo job `pending` do mesmo asset, na ordem de `jobcontracts.Pipeline`, no stream indicado por `jobcontracts.StreamFor`. O encadeamento SHALL continuar depois do `transcode-hls`. Jobs fora do pipeline, como `export-clip`, SHALL não encadear nada.

#### Scenario: Metadata falha e o pipeline continua
- **WHEN** `extract-metadata` falha
- **THEN** `generate-thumbnail` SHALL ser enfileirado e o asset SHALL continuar `processing`

#### Scenario: Próximo job não pode ser enfileirado
- **WHEN** o enfileiramento do próximo job falha
- **THEN** esse job SHALL ficar `failed` com a mensagem do erro e o asset SHALL ficar `failed`

#### Scenario: Depois do HLS
- **WHEN** `transcode-hls` termina
- **THEN** `extract-audio` SHALL ser enfileirado em `jobs`; depois dele, `transcribe-audio`, `analyze-transcript`, `suggest-clips` e `track-faces` SHALL ser enfileirados em `jobs:ai`, um de cada vez
