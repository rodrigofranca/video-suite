# worker-skeleton Specification

## Purpose
Define o esqueleto do worker em Go que consome jobs do Redis Streams, despacha cada mensagem pelo tipo de job e confirma o processamento.

## Requirements

### Requirement: Worker conecta ao Redis e consome jobs via XREADGROUP
O binário `apps/worker` SHALL conectar ao Redis na inicialização, criar consumer group `workers` no stream `jobs` se não existir, e entrar em loop de consumo via XREADGROUP.

#### Scenario: Worker inicia sem erro com Redis disponível
- **WHEN** `make dev-worker` é executado com Redis rodando
- **THEN** o worker SHALL iniciar, logar conexão estabelecida e entrar em loop de consumo

#### Scenario: Worker falha rápido sem Redis
- **WHEN** Redis não está disponível na inicialização
- **THEN** o worker SHALL logar o erro e encerrar com código de saída não-zero

### Requirement: Worker despacha por tipo de job
Ao consumir uma mensagem do stream, o worker SHALL ler o campo `type` e despachar para o handler correspondente. Tipos sem handler registrado SHALL ser logados como `unsupported` e ackados para não bloquear a fila.

#### Scenario: Job com tipo conhecido é despachado
- **WHEN** mensagem com `type = "extract-metadata"` é consumida
- **THEN** o handler registrado para `extract-metadata` SHALL ser chamado com o `job_id` e o payload deserializado em `ExtractMetadataInput`

#### Scenario: Job com tipo desconhecido é ackado com log
- **WHEN** mensagem com `type = "unknown-type"` é consumida
- **THEN** o worker SHALL logar `unsupported job type: unknown-type` e executar XACK

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

### Requirement: Worker retoma mensagens pendentes ao reiniciar
Ao iniciar, o worker SHALL reprocessar as mensagens já entregues ao seu nome de consumidor e ainda não ackadas, antes de ler mensagens novas. O nome do consumidor SHALL vir de `WORKER_NAME`, com o hostname como padrão.

#### Scenario: Job interrompido por queda é reprocessado
- **WHEN** o worker cai depois de receber uma mensagem e antes do XACK, e é reiniciado com o mesmo nome
- **THEN** o worker SHALL processar essa mensagem e executar XACK antes de consumir mensagens novas

### Requirement: Encerramento gracioso do worker
O worker SHALL encerrar ao receber SIGINT ou SIGTERM, ackando um job já concluído mesmo durante o encerramento.

#### Scenario: Ctrl+C encerra o worker
- **WHEN** o worker recebe SIGINT enquanto aguarda mensagens
- **THEN** SHALL logar `worker shut down` e encerrar com código zero em poucos segundos

### Requirement: Worker verifica dependências na inicialização
O worker SHALL exigir `REDIS_URL`, `DATABASE_URL` e as variáveis `S3_*` obrigatórias, encontrar o `ffmpeg` no `PATH` e conferir Redis, PostgreSQL e o bucket antes de consumir jobs.

#### Scenario: FFmpeg ausente
- **WHEN** o worker inicia numa máquina sem FFmpeg
- **THEN** SHALL logar `ffmpeg not found in PATH` e encerrar com código não-zero
