# Spec Delta

## MODIFIED Requirements

### Requirement: Job é ackado após execução do handler
O worker SHALL executar XACK após o handler retornar, independente de sucesso ou erro. Erros do handler SHALL ser logados mas não devem travar o loop. A única exceção é um job que pede nova tentativa (`queue.ErrRetry`), como um job interrompido pelo encerramento do worker.

#### Scenario: Handler com erro não trava o worker
- **WHEN** handler retorna erro
- **THEN** o worker SHALL logar o erro, executar XACK e continuar processando o próximo job

#### Scenario: Panic no handler não derruba o worker
- **WHEN** o handler entra em panic
- **THEN** o worker SHALL tratar o panic como erro, logar, executar XACK e continuar

#### Scenario: Mensagem malformada é descartada com log
- **WHEN** a mensagem não tem `job_id`, `type` ou `payload` válido
- **THEN** o worker SHALL logar `invalid job message` e executar XACK

#### Scenario: Job que pede nova tentativa fica pendente
- **WHEN** o processamento retorna `queue.ErrRetry`
- **THEN** o worker SHALL não executar XACK, seguir para a próxima mensagem e reprocessar essa no próximo start, uma única vez

### Requirement: Handlers do MVP1 são stubs que logam e retornam nil
Os handlers de `extract-metadata` e `generate-thumbnail` SHALL continuar como stubs que logam `stub: <type> called for asset <id> (not implemented yet)` no log do job e retornam nil. O handler de `transcode-hls` segue a spec `hls-transcoding`.

#### Scenario: Stub executa sem erro
- **WHEN** handler stub é chamado com payload válido
- **THEN** o handler SHALL logar a mensagem stub e retornar nil

## ADDED Requirements

### Requirement: Worker verifica dependências na inicialização
O worker SHALL exigir `REDIS_URL`, `DATABASE_URL` e as variáveis `S3_*` obrigatórias, encontrar o `ffmpeg` no `PATH` e conferir Redis, PostgreSQL e o bucket antes de consumir jobs.

#### Scenario: FFmpeg ausente
- **WHEN** o worker inicia numa máquina sem FFmpeg
- **THEN** SHALL logar `ffmpeg not found in PATH` e encerrar com código não-zero
