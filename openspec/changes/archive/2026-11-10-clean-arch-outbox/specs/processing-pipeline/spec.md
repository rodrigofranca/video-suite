## MODIFIED Requirements

### Requirement: Encadeamento do pipeline
Quando um job termina, com sucesso ou falha, o relay (ADR 0034) SHALL pôr na fila o próximo job `pending` do mesmo asset, na ordem de `jobcontracts.Pipeline`, no stream indicado por `jobcontracts.StreamFor`. O encadeamento SHALL continuar depois do `transcode-hls`. Jobs fora do pipeline, como `export-clip`, SHALL não encadear nada. Se a fila estiver fora do ar, o próximo job SHALL continuar `pending` e ir para a fila quando ela voltar.

#### Scenario: Metadata falha e o pipeline continua
- **WHEN** `extract-metadata` falha
- **THEN** `generate-thumbnail` SHALL ser enfileirado e o asset SHALL continuar `processing`

#### Scenario: Próximo job não pode ser enfileirado
- **WHEN** o próximo job não pode ser mandado ao Redis
- **THEN** ele SHALL continuar `pending`, o asset SHALL continuar `processing`, e o job SHALL ir para a fila numa nova tentativa

#### Scenario: Depois do HLS
- **WHEN** `transcode-hls` termina
- **THEN** `extract-audio` SHALL ser enfileirado em `jobs`; depois dele, `transcribe-audio`, `analyze-transcript`, `suggest-clips` e `track-faces` SHALL ser enfileirados em `jobs:ai`, um de cada vez

### Requirement: Status do asset pelo job HLS
O relay SHALL marcar o asset `ready` quando `transcode-hls` termina com sucesso e `failed` quando ele falha. Os passos seguintes SHALL não mudar o status do asset.

#### Scenario: Pipeline completo
- **WHEN** os jobs até o `transcode-hls` terminam com sucesso
- **THEN** o asset SHALL ficar `ready`, mesmo com a extração de áudio, a transcrição e a análise ainda pendentes

#### Scenario: Análise com falha
- **WHEN** `analyze-transcript` falha
- **THEN** o asset SHALL continuar `ready`

### Requirement: Execução única e retomada
Uma entrega repetida de um job `done` ou `failed` SHALL ser ignorada, sem gravar outro `job.finished`. Um job interrompido pelo encerramento do worker SHALL continuar `running`, sem ack, e rodar de novo quando o worker reiniciar.

#### Scenario: Entrega duplicada
- **WHEN** a mesma mensagem é processada duas vezes
- **THEN** o handler SHALL rodar só uma vez

#### Scenario: Encerramento no meio do job
- **WHEN** o worker recebe SIGTERM durante um job
- **THEN** o job SHALL continuar `running`, a mensagem SHALL ficar pendente e o job SHALL ser executado de novo no próximo start
