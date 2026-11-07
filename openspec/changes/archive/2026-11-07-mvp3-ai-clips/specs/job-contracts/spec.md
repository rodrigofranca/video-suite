# Spec Delta

## MODIFIED Requirements

### Requirement: Jobs de áudio e transcrição
O pacote SHALL definir `extract-audio` (`ExtractAudioInput`: `asset_id`, `storage_key`, `audio_key`) e `transcribe-audio` (`TranscribeAudioInput`: `asset_id`, `audio_key`, `language` opcional), e o `Pipeline` SHALL ser `extract-metadata`, `generate-thumbnail`, `transcode-hls`, `extract-audio`, `transcribe-audio`, `analyze-transcript`, `suggest-clips`.

#### Scenario: Próximo passo depois do HLS
- **WHEN** `NextJobType("transcode-hls")` é chamado
- **THEN** SHALL retornar `extract-audio`

#### Scenario: Próximo passo depois da transcrição
- **WHEN** `NextJobType("transcribe-audio")` é chamado
- **THEN** SHALL retornar `analyze-transcript`, `NextJobType("analyze-transcript")` SHALL retornar `suggest-clips` e `NextJobType("suggest-clips")` SHALL indicar que não há próximo passo

### Requirement: Roteamento por stream
`StreamFor` SHALL devolver `jobs:ai` para os jobs do worker de IA (`transcribe-audio`, `analyze-transcript` e `suggest-clips`) e `jobs` para os demais, inclusive `export-clip`, e todo enfileiramento SHALL usar o stream devolvido por ela.

#### Scenario: Transcrição vai para o worker de IA
- **WHEN** a mensagem de um job `transcribe-audio` é enfileirada
- **THEN** ela SHALL ir para o stream `jobs:ai`

#### Scenario: Análise vai para o worker de IA
- **WHEN** a mensagem de um job `analyze-transcript` é enfileirada
- **THEN** ela SHALL ir para o stream `jobs:ai`

#### Scenario: Exportação vai para o worker Go
- **WHEN** a mensagem de um job `export-clip` é enfileirada
- **THEN** ela SHALL ir para o stream `jobs`

## ADDED Requirements

### Requirement: Jobs de cortes
O pacote SHALL definir `suggest-clips` (`SuggestClipsInput`: `asset_id`) no fim do `Pipeline`, e `export-clip` (`ExportClipInput`: `asset_id`, `clip_id`, `storage_key`, `format`, `captions`, `output_key`) fora dele, com os formatos `original`, `vertical` e `vertical-blur` em `ClipFormats`.

#### Scenario: Exportação fora do pipeline
- **WHEN** `NextJobType("export-clip")` é chamado
- **THEN** SHALL indicar que não há próximo passo
