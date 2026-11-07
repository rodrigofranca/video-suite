# processing-pipeline Specification

## Purpose
Define como o worker executa cada job do pipeline de um asset: status no banco, logs, limites de tempo, encadeamento do próximo passo e o status final do asset.

## Requirements

### Requirement: Ciclo de vida do job no banco
Para cada mensagem, o worker SHALL carregar o job do PostgreSQL, marcá-lo `running` com `started_at`, executar o handler e marcá-lo `done` (progresso 100) ou `failed` com `error_message`, sempre com `completed_at`.

#### Scenario: Job com erro
- **WHEN** o handler retorna erro
- **THEN** o job SHALL ficar `failed` com a mensagem do erro e o log do job SHALL ter uma linha `error`

#### Scenario: Job inexistente
- **WHEN** a mensagem aponta para um job que não existe no banco
- **THEN** o worker SHALL logar o erro e ackar a mensagem

### Requirement: Logs do job
Tudo que um handler loga SHALL ser gravado em `job_logs` com nível `info`, `warning` ou `error`, além de aparecer na saída do worker.

#### Scenario: Linhas de início e fim
- **WHEN** um job roda com sucesso
- **THEN** seus logs SHALL incluir `job started` e `job finished`

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

### Requirement: Limite de tempo e panics
Cada job SHALL ter limite de 30 minutos. Estourar o limite ou entrar em panic SHALL marcar o job como `failed`, sem derrubar o worker.

#### Scenario: Tempo esgotado
- **WHEN** um handler passa do limite
- **THEN** o job SHALL ficar `failed` com a mensagem `timed out after 30m0s`

### Requirement: Execução única e retomada
Uma entrega repetida de um job `done` ou `failed` SHALL ser ignorada. Um job interrompido pelo encerramento do worker SHALL continuar `running`, sem ack, e rodar de novo quando o worker reiniciar.

#### Scenario: Entrega duplicada
- **WHEN** a mesma mensagem é processada duas vezes
- **THEN** o handler SHALL rodar só uma vez

#### Scenario: Encerramento no meio do job
- **WHEN** o worker recebe SIGTERM durante um job
- **THEN** o job SHALL continuar `running`, a mensagem SHALL ficar pendente e o job SHALL ser executado de novo no próximo start

### Requirement: Progresso dos jobs
O worker SHALL gravar o progresso que um handler reporta em `processing_jobs.progress` enquanto o job está `running`, no máximo uma vez por segundo e só quando o valor muda. Um job `done` SHALL ficar com progresso 100.

#### Scenario: Valores próximos
- **WHEN** o handler reporta 10 e logo depois 20
- **THEN** o banco SHALL mostrar 10 até passar um segundo

#### Scenario: Job pendente
- **WHEN** o progresso é gravado para um job que não está `running`
- **THEN** o valor SHALL ser ignorado
