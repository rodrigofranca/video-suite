# audio-extraction Specification

## Purpose
Define o job `extract-audio`, que prepara o áudio do vídeo para a transcrição sem que o worker de IA precise baixar o vídeo inteiro.

## Requirements

### Requirement: Áudio para transcrição
O job SHALL ler o original por URL assinada e gravar a primeira faixa de áudio como FLAC mono a 16 kHz em `assets/{id}/audio/audio.flac`, registrando um artefato `audio`. A chave de destino SHALL ser exatamente a do asset.

#### Scenario: Vídeo com áudio
- **WHEN** o job roda num vídeo com faixa de áudio
- **THEN** o arquivo gravado SHALL ser FLAC, 16000 Hz, 1 canal

#### Scenario: Chave de outro asset
- **WHEN** `audio_key` não é a chave de áudio do próprio asset
- **THEN** o job SHALL falhar com `invalid input`

### Requirement: Vídeo sem áudio
Um vídeo sem faixa de áudio SHALL não ser erro: o job SHALL terminar com sucesso, registrar no log que não há o que transcrever e não gravar arquivo.

#### Scenario: Vídeo mudo
- **WHEN** o vídeo não tem áudio
- **THEN** o job SHALL ficar `done` com o log `no audio track` e nenhum arquivo SHALL existir em `assets/{id}/audio/`
