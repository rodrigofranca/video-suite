# hls-transcoding Specification

## Purpose
Define o job `transcode-hls`, que converte o vídeo de origem num stream HLS de uma única qualidade, reproduzível em qualquer navegador.

## Requirements

### Requirement: Conversão para HLS com FFmpeg
O job SHALL baixar o source, medi-lo e executar o FFmpeg uma vez, com lista de argumentos (sem shell), gerando H.264 (`yuv420p`, dimensões pares) com AAC estéreo opcional, em segmentos MPEG-TS de 6 segundos, numa pasta por qualidade com sua playlist VOD `index.m3u8`, e uma playlist mestre `index.m3u8` com as qualidades.

#### Scenario: Vídeo de 13 segundos
- **WHEN** um MP4 de 13 segundos em 1280x720 é convertido
- **THEN** cada qualidade SHALL ter 3 segmentos, com os mesmos tempos em todas, e cada playlist SHALL terminar com `#EXT-X-ENDLIST`

#### Scenario: Dimensões ímpares
- **WHEN** o vídeo tem largura ou altura ímpar
- **THEN** a conversão SHALL funcionar, ajustando para dimensões pares

### Requirement: Publicação do resultado
O job SHALL enviar os segmentos, depois a playlist de cada qualidade e por último a playlist mestre para `assets/{id}/hls/`, apagar dessa pasta os arquivos que não escreveu e registrar um artefato `hls` apontando para a playlist mestre, com o número de segmentos e as qualidades nos metadados.

#### Scenario: Artefato criado
- **WHEN** a conversão termina
- **THEN** SHALL existir um artefato `hls` com `storage_key` `assets/{id}/hls/index.m3u8`, ligado ao job

#### Scenario: Codificação anterior
- **WHEN** o HLS roda de novo num vídeo que tinha a qualidade única antiga
- **THEN** os segmentos antigos em `hls/` SHALL ser apagados e o log SHALL dizer quantos

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

### Requirement: Progresso da conversão
Durante a conversão, o job SHALL reportar o progresso como a fração da duração já escrita pelo FFmpeg, de 0 a 99, chegando a 100 só quando o job termina. A duração SHALL vir do próprio arquivo baixado, sem depender do job de metadata.

#### Scenario: Progresso crescente
- **WHEN** um vídeo de 8 segundos é convertido
- **THEN** os valores reportados SHALL crescer, ficar entre 0 e 99 e o último SHALL ser pelo menos 80

### Requirement: Qualidades
As qualidades SHALL seguir o lado menor do vídeo exibido: o tamanho dele, no máximo 1080p, e cada degrau de 720p, 480p e 360p abaixo, menos os que ficam a menos de 10% do de cima, mantendo a proporção. Cada qualidade SHALL ter CRF 21 com teto de bitrate (5000, 2800, 1400 e 800 kbps, proporcional entre e abaixo deles), e a playlist mestre SHALL anunciar `BANDWIDTH` e `RESOLUTION` de cada uma.

#### Scenario: Vídeo 1080p
- **WHEN** o vídeo tem 1920x1080
- **THEN** SHALL haver 1080p, 720p, 480p e 360p

#### Scenario: Vídeo em pé
- **WHEN** o vídeo tem 1080x1920
- **THEN** o 720p SHALL ter 720x1280

#### Scenario: Vídeo pequeno
- **WHEN** o vídeo tem 321x241
- **THEN** SHALL haver só uma qualidade, 320x240
