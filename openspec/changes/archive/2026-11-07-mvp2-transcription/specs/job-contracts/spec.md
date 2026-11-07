# Spec Delta

## ADDED Requirements

### Requirement: Jobs de áudio e transcrição
O pacote SHALL definir `extract-audio` (`ExtractAudioInput`: `asset_id`, `storage_key`, `audio_key`) e `transcribe-audio` (`TranscribeAudioInput`: `asset_id`, `audio_key`, `language` opcional), e o `Pipeline` SHALL ser `extract-metadata`, `generate-thumbnail`, `transcode-hls`, `extract-audio`, `transcribe-audio`.

#### Scenario: Próximo passo depois do HLS
- **WHEN** `NextJobType("transcode-hls")` é chamado
- **THEN** SHALL retornar `extract-audio`

### Requirement: Roteamento por stream
`StreamFor` SHALL devolver `jobs:ai` para os jobs do worker de IA (`transcribe-audio`) e `jobs` para os demais, e todo enfileiramento SHALL usar o stream devolvido por ela.

#### Scenario: Transcrição vai para o worker de IA
- **WHEN** a mensagem de um job `transcribe-audio` é enfileirada
- **THEN** ela SHALL ir para o stream `jobs:ai`

### Requirement: Arquivo de contratos compartilhado
O pacote SHALL manter `testdata/contracts.json` com tipos, ordem do pipeline, rotas, streams, consumer groups, campos e exemplos de entrada, e um teste SHALL falhar quando o arquivo não corresponder ao código.

#### Scenario: Arquivo desatualizado
- **WHEN** o código muda sem regenerar o arquivo
- **THEN** o teste SHALL falhar indicando `go test ./packages/job-contracts -update`
