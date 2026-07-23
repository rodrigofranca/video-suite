## ADDED Requirements

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
- **THEN** o handler `ExtractMetadataHandler` SHALL ser chamado com o payload deserializado

#### Scenario: Job com tipo desconhecido é ackado com log
- **WHEN** mensagem com `type = "unknown-type"` é consumida
- **THEN** o worker SHALL logar `unsupported job type: unknown-type` e executar XACK

### Requirement: Job é ackado após execução do handler
O worker SHALL executar XACK após o handler retornar, independente de sucesso ou erro. Erros do handler SHALL ser logados mas não devem travar o loop.

#### Scenario: Handler com erro não trava o worker
- **WHEN** handler retorna erro
- **THEN** o worker SHALL logar o erro, executar XACK e continuar processando o próximo job

### Requirement: Handlers do MVP1 são stubs que logam e retornam nil
No escopo da fundação, os handlers para `extract-metadata`, `generate-thumbnail` e `transcode-hls` SHALL existir como stubs que logam `stub: <type> called for asset <id>` e retornam nil.

#### Scenario: Stub executa sem erro
- **WHEN** handler stub é chamado com payload válido
- **THEN** o handler SHALL logar a mensagem stub e retornar nil
