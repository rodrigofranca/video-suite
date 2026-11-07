# hls-transcoding Specification

## Purpose
Define o job `transcode-hls`, que converte o vídeo de origem num stream HLS de uma única qualidade, reproduzível em qualquer navegador.

## Requirements

### Requirement: Conversão para HLS com FFmpeg
O job SHALL baixar o source, executar o FFmpeg com lista de argumentos (sem shell) e gerar H.264 (`yuv420p`, dimensões pares) com AAC estéreo opcional, em segmentos MPEG-TS de 6 segundos e playlist VOD `index.m3u8`.

#### Scenario: Vídeo de 13 segundos
- **WHEN** um MP4 de 13 segundos é convertido
- **THEN** SHALL haver 3 segmentos e a playlist SHALL terminar com `#EXT-X-ENDLIST`

#### Scenario: Dimensões ímpares
- **WHEN** o vídeo tem largura ou altura ímpar
- **THEN** a conversão SHALL funcionar, ajustando para dimensões pares

### Requirement: Publicação do resultado
O job SHALL enviar os segmentos e depois a playlist para `assets/{id}/hls/` e registrar um artefato `hls` apontando para a playlist, com o número de segmentos nos metadados.

#### Scenario: Artefato criado
- **WHEN** a conversão termina
- **THEN** SHALL existir um artefato `hls` com `storage_key` `assets/{id}/hls/index.m3u8`, ligado ao job

### Requirement: Erros do FFmpeg visíveis
As linhas que o FFmpeg imprime (avisos e erros) SHALL ir para o log do job, e uma falha SHALL ter mensagem de erro com as últimas linhas do FFmpeg.

#### Scenario: Arquivo corrompido
- **WHEN** o source não é um vídeo válido
- **THEN** o job SHALL falhar com uma mensagem que começa com `ffmpeg failed` e inclui a explicação do FFmpeg

### Requirement: Entrada validada
O job SHALL recusar `asset_id` que não seja UUID ou `storage_key` vazia, antes de tocar no storage.

#### Scenario: asset_id malicioso
- **WHEN** `asset_id` é `../../etc`
- **THEN** o job SHALL falhar com `invalid input`
