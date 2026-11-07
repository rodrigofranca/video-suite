# Spec Delta

## MODIFIED Requirements

### Requirement: Organização por asset
Os arquivos SHALL ficar em `assets/{assetId}/source/input.mp4` (original), `assets/{assetId}/hls/` (playlist `index.m3u8` e segmentos), `assets/{assetId}/thumbnails/thumbnail.jpg` (miniatura), `assets/{assetId}/audio/audio.flac` (áudio para transcrição) e `assets/{assetId}/subtitles/{idioma}.vtt` (legendas), com chaves geradas a partir do ID.

#### Scenario: Caminho do original
- **WHEN** um asset é criado por upload
- **THEN** o original SHALL estar em `assets/{assetId}/source/input.mp4`

#### Scenario: Caminho da miniatura
- **WHEN** o job de miniatura termina
- **THEN** a imagem SHALL estar em `assets/{assetId}/thumbnails/thumbnail.jpg`

#### Scenario: Caminho das legendas
- **WHEN** a transcrição em inglês termina
- **THEN** as legendas SHALL estar em `assets/{assetId}/subtitles/en.vtt`

### Requirement: Só as saídas derivadas são públicas
O bucket SHALL permitir leitura anônima apenas em `assets/*/hls/*`, `assets/*/thumbnails/*` e `assets/*/subtitles/*`. O arquivo original e o áudio extraído SHALL exigir credenciais.

#### Scenario: Leitura anônima
- **WHEN** um navegador sem credenciais pede a playlist, a miniatura e o original de um asset
- **THEN** a playlist e a miniatura SHALL responder 200 e o original SHALL responder 403
