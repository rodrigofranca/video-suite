# object-storage Specification

## Purpose
Define como os arquivos de mídia são organizados no storage compatível com S3 e quais deles podem ser lidos publicamente pelo navegador.

## Requirements

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

### Requirement: Apenas a API S3
O acesso ao storage SHALL usar somente a API S3 (endereçamento por caminho, checksums só quando exigidos), configurada por `S3_ENDPOINT`, `S3_REGION`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET` e `S3_PUBLIC_URL`.

#### Scenario: Upload grande
- **WHEN** um arquivo maior que o tamanho de parte (16 MB) é enviado
- **THEN** ele SHALL ser gravado por upload multipart e baixado de volta idêntico

### Requirement: Só as saídas derivadas são públicas
O bucket SHALL permitir leitura anônima apenas em `assets/*/hls/*`, `assets/*/thumbnails/*` e `assets/*/subtitles/*`. O arquivo original e o áudio extraído SHALL exigir credenciais.

#### Scenario: Leitura anônima
- **WHEN** um navegador sem credenciais pede a playlist, a miniatura e o original de um asset
- **THEN** a playlist e a miniatura SHALL responder 200 e o original SHALL responder 403

### Requirement: Leitura temporária do original
O storage SHALL gerar URLs assinadas de leitura com validade definida, que aceitam requisições parciais (HTTP Range).

#### Scenario: Leitura de um trecho
- **WHEN** uma URL assinada é pedida com `Range: bytes=2-5`
- **THEN** a resposta SHALL ser 206 com apenas esses bytes

### Requirement: Cortes privados
Os cortes exportados SHALL ficar em `assets/{id}/clips/{clipId}/{formato}.mp4`, sem leitura anônima, e SHALL ser entregues só por links assinados gerados pela API.

#### Scenario: Leitura anônima de um corte
- **WHEN** um navegador sem credenciais pede o arquivo de um corte direto no storage
- **THEN** o storage SHALL recusar

### Requirement: Apagar uma pasta
O cliente SHALL apagar todos os objetos sob um prefixo e dizer quantos apagou, aceitando só a pasta de um asset ou uma pasta dentro dela (`assets/<id>/...`, terminando em barra, sem `..`).

#### Scenario: Prefixo perigoso
- **WHEN** o prefixo é vazio, `assets/`, `assets/x` ou fora de `assets/`
- **THEN** o cliente SHALL recusar sem apagar nada

#### Scenario: Pasta de um corte
- **WHEN** a pasta de um corte é apagada
- **THEN** só os arquivos desse corte SHALL sumir

### Requirement: Listar uma pasta
O cliente SHALL listar as chaves dos objetos sob um prefixo, página a página.

#### Scenario: Arquivos antigos do HLS
- **WHEN** o job lista `assets/{id}/hls/` depois de enviar uma nova codificação
- **THEN** SHALL receber também os arquivos de codificações anteriores, para apagá-los
