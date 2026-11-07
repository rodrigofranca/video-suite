# Spec Delta

## MODIFIED Requirements

### Requirement: Organização por asset
Os arquivos SHALL ficar em `assets/{assetId}/source/input.mp4` (original), `assets/{assetId}/hls/` (playlist `index.m3u8` e segmentos) e `assets/{assetId}/thumbnails/thumbnail.jpg` (miniatura), com chaves geradas a partir do ID.

#### Scenario: Caminho do original
- **WHEN** um asset é criado por upload
- **THEN** o original SHALL estar em `assets/{assetId}/source/input.mp4`

#### Scenario: Caminho da miniatura
- **WHEN** o job de miniatura termina
- **THEN** a imagem SHALL estar em `assets/{assetId}/thumbnails/thumbnail.jpg`

## REMOVED Requirements

### Requirement: Só o HLS é público
**Reason**: As miniaturas também precisam ser lidas pelo navegador (ADR 0013).
**Migration**: Substituído pelo requisito "Só as saídas derivadas são públicas".

## ADDED Requirements

### Requirement: Só as saídas derivadas são públicas
O bucket SHALL permitir leitura anônima apenas em `assets/*/hls/*` e `assets/*/thumbnails/*`. O arquivo original SHALL exigir credenciais.

#### Scenario: Leitura anônima
- **WHEN** um navegador sem credenciais pede a playlist, a miniatura e o original de um asset
- **THEN** a playlist e a miniatura SHALL responder 200 e o original SHALL responder 403

### Requirement: Leitura temporária do original
O storage SHALL gerar URLs assinadas de leitura com validade definida, que aceitam requisições parciais (HTTP Range).

#### Scenario: Leitura de um trecho
- **WHEN** uma URL assinada é pedida com `Range: bytes=2-5`
- **THEN** a resposta SHALL ser 206 com apenas esses bytes
