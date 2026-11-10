## MODIFIED Requirements

### Requirement: Arquivo de contratos compartilhado
O pacote SHALL manter `testdata/contracts.json` com tipos, a tabela de tipos de job (`jobs`: nome, rótulo, worker e se pode rodar de novo à mão), ordem do pipeline, rotas, streams, consumer groups, campos e exemplos de entrada, e um teste SHALL falhar quando o arquivo não corresponder ao código.

#### Scenario: Arquivo desatualizado
- **WHEN** o código muda sem regenerar o arquivo
- **THEN** o teste SHALL falhar indicando `go test ./packages/job-contracts -update`

## ADDED Requirements

### Requirement: Tabela de tipos de job
O pacote SHALL definir `JobTypes`, uma entrada por tipo de job com nome, rótulo, worker (`media` ou `ai`) e se pode rodar de novo à mão, primeiro os passos do `Pipeline`, na ordem. `StreamFor` SHALL usar o worker da tabela. Todo tipo citado em `Pipeline`, `RerunWith` e `After` SHALL estar na tabela, e cada worker SHALL ter um handler para cada tipo seu, conferido em teste.

#### Scenario: Tipo novo sem handler
- **WHEN** um tipo de job do worker de mídia é posto na tabela sem handler no worker
- **THEN** o teste do worker de mídia SHALL falhar
