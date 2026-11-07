# thumbnail-generation Specification

## Purpose
Define o job `generate-thumbnail`, que salva um quadro do vídeo como imagem para a lista de vídeos e para a capa do player.

## Requirements

### Requirement: Escolha do quadro
O job SHALL usar o segundo pedido em `AtSecond`; quando a duração conhecida for menor ou igual a ele, SHALL usar o meio do vídeo. Se nenhum quadro for gerado, SHALL tentar o primeiro quadro.

#### Scenario: Vídeo mais curto que o segundo pedido
- **WHEN** `AtSecond` é 3 e o vídeo tem 2 segundos
- **THEN** a miniatura SHALL ser o quadro em 1 segundo

#### Scenario: Duração desconhecida
- **WHEN** o job de metadata falhou e o segundo pedido passa do fim do vídeo
- **THEN** a miniatura SHALL ser o primeiro quadro e o log SHALL dizer `using the first frame`

### Requirement: Formato e publicação
A miniatura SHALL ser um JPEG com no máximo 1280 pixels de largura, mantendo a proporção, gravado em `assets/{id}/thumbnails/thumbnail.jpg` e registrado como artefato `thumbnail` com segundo, largura e altura.

#### Scenario: Vídeo 1080p
- **WHEN** o vídeo é 1920x1080
- **THEN** a miniatura SHALL ter 1280x720 e ser lida sem credenciais como `image/jpeg`

### Requirement: Leitura parcial do original
O job SHALL ler o original por URL assinada, que SHALL aparecer como `<source>` no comando registrado no log.

#### Scenario: Comando no log
- **WHEN** o job roda
- **THEN** o log SHALL mostrar o comando FFmpeg com `-i <source>`
