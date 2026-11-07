# Spec Delta

## Purpose
Define o job `transcribe-audio`, que transforma a fala do vídeo em texto com tempos e em legendas WebVTT para o player.

## ADDED Requirements

### Requirement: Transcrição com Whisper
O job SHALL transcrever o áudio extraído com faster-whisper, usando o modelo de `WHISPER_MODEL` (padrão `base`), detectando o idioma quando a entrada não informa um, e SHALL gravar em `transcripts` o idioma, o texto, os trechos com início e fim e o modelo usado. Rodar de novo SHALL substituir a transcrição anterior.

#### Scenario: Fala em inglês
- **WHEN** o áudio contém a frase sintética "Hello and welcome to the ultra media platform..."
- **THEN** a transcrição SHALL ter idioma `en` e conter "welcome" e "platform"

#### Scenario: Sem áudio
- **WHEN** não existe áudio extraído para o asset
- **THEN** o job SHALL terminar com sucesso, com o log `no audio to transcribe`, e nenhuma transcrição SHALL ser gravada

### Requirement: Legendas WebVTT
Quando houver fala, o job SHALL gravar legendas WebVTT em `assets/{id}/subtitles/{idioma}.vtt`, com `Content-Type` `text/vtt`, e registrar um artefato `subtitles` com o idioma. Cada legenda SHALL durar no máximo 6 segundos e ter no máximo 84 caracteres em até duas linhas.

#### Scenario: Legendas públicas
- **WHEN** a transcrição termina com fala
- **THEN** o arquivo de legendas SHALL ser lido sem credenciais e começar com `WEBVTT`

### Requirement: Progresso e entrada validada
O job SHALL reportar progresso pela posição do último trecho transcrito em relação à duração do áudio, e SHALL recusar `asset_id` que não seja UUID ou `audio_key` diferente da chave de áudio do asset.

#### Scenario: Chave de outro asset
- **WHEN** `audio_key` aponta para outro asset
- **THEN** o job SHALL falhar com `invalid input`
