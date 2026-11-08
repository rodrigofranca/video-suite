# job-contracts Specification

## Purpose
Define o pacote compartilhado de contratos de job, com os tipos de job e as entradas tipadas que a API enfileira e o worker consome.

## Requirements

### Requirement: Pacote job-contracts com structs tipadas
O módulo `packages/job-contracts` SHALL definir structs Go para os 3 tipos de job do MVP1: `ExtractMetadataInput`, `GenerateThumbnailInput`, `TranscodeHLSInput`.

#### Scenario: Pacote compila de forma isolada
- **WHEN** `go build ./...` é executado em `packages/job-contracts`
- **THEN** o pacote SHALL compilar sem dependências externas além da stdlib

#### Scenario: Structs têm campos mínimos documentados
- **WHEN** o código de `job-contracts` é inspecionado
- **THEN** `ExtractMetadataInput` SHALL ter pelo menos `AssetID` e `StorageKey`
- **THEN** `GenerateThumbnailInput` SHALL ter pelo menos `AssetID`, `StorageKey` e `AtSecond`
- **THEN** `TranscodeHLSInput` SHALL ter pelo menos `AssetID` e `StorageKey`

### Requirement: Constantes para tipos de job
O pacote SHALL exportar constantes string para os tipos de job: `JobTypeExtractMetadata`, `JobTypeGenerateThumbnail`, `JobTypeTranscodeHLS`.

#### Scenario: Constantes usáveis por API e Worker
- **WHEN** API ou Worker importam `job-contracts`
- **THEN** as constantes SHALL estar disponíveis sem cast ou string literal hardcoded

### Requirement: job-contracts importável por API e Worker via Go Workspace
API e Worker SHALL importar `job-contracts` sem `replace` directives no próprio `go.mod`, usando apenas `go.work` na raiz.

#### Scenario: Import resolve via go.work
- **WHEN** `go build ./...` é executado em `apps/api` ou `apps/worker`
- **THEN** o import de `job-contracts` SHALL resolver sem erro

### Requirement: Formato da mensagem na fila
O pacote SHALL definir o formato das mensagens do stream `jobs` (campos `job_id`, `type` e `payload` com o JSON da struct de entrada), as constantes do stream e do consumer group, e as funções para montar (`NewMessage`, `Values`) e ler (`ParseMessage`, `Decode`) uma mensagem.

#### Scenario: Mensagem faz ida e volta sem perda
- **WHEN** a API monta uma mensagem com `NewMessage` e o worker a lê com `ParseMessage` e `Decode`
- **THEN** o worker SHALL obter o mesmo `job_id`, o mesmo `type` e a mesma struct de entrada

#### Scenario: Mensagem incompleta é rejeitada
- **WHEN** `ParseMessage` recebe campos sem `job_id`, sem `type`, sem `payload` ou com `payload` que não é JSON válido
- **THEN** SHALL retornar erro

### Requirement: Jobs de áudio e transcrição
O pacote SHALL definir `extract-audio` (`ExtractAudioInput`: `asset_id`, `storage_key`, `audio_key`) e `transcribe-audio` (`TranscribeAudioInput`: `asset_id`, `audio_key`, `language` opcional), e o `Pipeline` SHALL ser `extract-metadata`, `generate-thumbnail`, `transcode-hls`, `extract-audio`, `transcribe-audio`, `analyze-transcript`, `suggest-clips`, `track-faces`.

#### Scenario: Próximo passo depois do HLS
- **WHEN** `NextJobType("transcode-hls")` é chamado
- **THEN** SHALL retornar `extract-audio`

#### Scenario: Próximo passo depois da transcrição
- **WHEN** `NextJobType("transcribe-audio")` é chamado
- **THEN** SHALL retornar `analyze-transcript`, `NextJobType("analyze-transcript")` SHALL retornar `suggest-clips`, `NextJobType("suggest-clips")` SHALL retornar `track-faces` e `NextJobType("track-faces")` SHALL indicar que não há próximo passo

### Requirement: Roteamento por stream
`StreamFor` SHALL devolver `jobs:ai` para os jobs do worker de IA (`transcribe-audio`, `analyze-transcript`, `suggest-clips` e `track-faces`) e `jobs` para os demais, inclusive `export-clip`, e todo enfileiramento SHALL usar o stream devolvido por ela.

#### Scenario: Transcrição vai para o worker de IA
- **WHEN** a mensagem de um job `transcribe-audio` é enfileirada
- **THEN** ela SHALL ir para o stream `jobs:ai`

#### Scenario: Análise vai para o worker de IA
- **WHEN** a mensagem de um job `analyze-transcript` é enfileirada
- **THEN** ela SHALL ir para o stream `jobs:ai`

#### Scenario: Exportação vai para o worker Go
- **WHEN** a mensagem de um job `export-clip` é enfileirada
- **THEN** ela SHALL ir para o stream `jobs`

#### Scenario: Rastreio de rostos vai para o worker de IA
- **WHEN** a mensagem de um job `track-faces` é enfileirada
- **THEN** ela SHALL ir para o stream `jobs:ai`

### Requirement: Arquivo de contratos compartilhado
O pacote SHALL manter `testdata/contracts.json` com tipos, ordem do pipeline, rotas, streams, consumer groups, campos e exemplos de entrada, e um teste SHALL falhar quando o arquivo não corresponder ao código.

#### Scenario: Arquivo desatualizado
- **WHEN** o código muda sem regenerar o arquivo
- **THEN** o teste SHALL falhar indicando `go test ./packages/job-contracts -update`

### Requirement: Job de análise da transcrição
O pacote SHALL definir `analyze-transcript` (`AnalyzeTranscriptInput`: `asset_id`). A transcrição SHALL ser lida do banco pelo worker, não enviada na mensagem.

#### Scenario: Entrada mínima
- **WHEN** a API cria o job `analyze-transcript` de um asset
- **THEN** a entrada SHALL ser `{"asset_id": "<id>"}`

### Requirement: Jobs de cortes
O pacote SHALL definir `suggest-clips` (`SuggestClipsInput`: `asset_id`) e `track-faces` (`TrackFacesInput`: `asset_id`, `storage_key`) no `Pipeline`, e `export-clip` (`ExportClipInput`: `asset_id`, `clip_id`, `storage_key`, `format`, `captions`, `caption_style`, `output_key`) fora dele, com os formatos `original`, `vertical`, `vertical-face` e `vertical-blur` em `ClipFormats`.

#### Scenario: Exportação fora do pipeline
- **WHEN** `NextJobType("export-clip")` é chamado
- **THEN** SHALL indicar que não há próximo passo

### Requirement: Jobs refeitos juntos
O pacote SHALL definir `RerunWith`, com os jobs que rodam de novo depois de um job refeito à mão porque usam o resultado dele: `suggest-clips` leva `track-faces`.

#### Scenario: Cortes refeitos
- **WHEN** `RerunWith["suggest-clips"]` é lido
- **THEN** SHALL ser `["track-faces"]`
