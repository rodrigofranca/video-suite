# Spec Delta

## MODIFIED Requirements

### Requirement: Jobs de áudio e transcrição
O pacote SHALL definir `extract-audio` (`ExtractAudioInput`: `asset_id`, `storage_key`, `audio_key`) e `transcribe-audio` (`TranscribeAudioInput`: `asset_id`, `audio_key`, `language` opcional), e o `Pipeline` SHALL ser `extract-metadata`, `generate-thumbnail`, `transcode-hls`, `extract-audio`, `transcribe-audio`, `analyze-transcript`.

#### Scenario: Próximo passo depois do HLS
- **WHEN** `NextJobType("transcode-hls")` é chamado
- **THEN** SHALL retornar `extract-audio`

#### Scenario: Próximo passo depois da transcrição
- **WHEN** `NextJobType("transcribe-audio")` é chamado
- **THEN** SHALL retornar `analyze-transcript`, e `NextJobType("analyze-transcript")` SHALL indicar que não há próximo passo

### Requirement: Roteamento por stream
`StreamFor` SHALL devolver `jobs:ai` para os jobs do worker de IA (`transcribe-audio` e `analyze-transcript`) e `jobs` para os demais, e todo enfileiramento SHALL usar o stream devolvido por ela.

#### Scenario: Transcrição vai para o worker de IA
- **WHEN** a mensagem de um job `transcribe-audio` é enfileirada
- **THEN** ela SHALL ir para o stream `jobs:ai`

#### Scenario: Análise vai para o worker de IA
- **WHEN** a mensagem de um job `analyze-transcript` é enfileirada
- **THEN** ela SHALL ir para o stream `jobs:ai`

## ADDED Requirements

### Requirement: Job de análise da transcrição
O pacote SHALL definir `analyze-transcript` (`AnalyzeTranscriptInput`: `asset_id`). A transcrição SHALL ser lida do banco pelo worker, não enviada na mensagem.

#### Scenario: Entrada mínima
- **WHEN** a API cria o job `analyze-transcript` de um asset
- **THEN** a entrada SHALL ser `{"asset_id": "<id>"}`
