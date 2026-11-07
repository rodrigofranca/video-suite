# metadata-extraction Specification

## Purpose
Define o job `extract-metadata`, que lê os metadados técnicos do vídeo com ffprobe e os guarda para o dashboard e para os outros jobs.

## Requirements

### Requirement: Metadados extraídos com ffprobe
O job SHALL rodar ffprobe no original e gravar em `technical_metadata` um documento `{"summary", "ffprobe"}`. O resumo SHALL ter duração, largura, altura, codecs de vídeo e áudio, bitrate, frame rate, container e tamanho.

#### Scenario: Vídeo de 8 segundos em 640x360
- **WHEN** o job roda num MP4 H.264 + AAC de 8 s em 640x360
- **THEN** o resumo SHALL trazer duração de cerca de 8 s, `640x360`, `h264` e `aac`

#### Scenario: Arquivo sem vídeo
- **WHEN** o arquivo não tem stream de vídeo
- **THEN** o job SHALL falhar com `the file has no video stream`

### Requirement: Leitura parcial do original
O job SHALL ler o original por uma URL assinada, sem baixar o arquivo inteiro. A URL SHALL ser substituída pela chave de storage no campo `format.filename` gravado e SHALL não aparecer em logs nem em mensagens de erro.

#### Scenario: Nada vaza
- **WHEN** o job termina, com sucesso ou falha
- **THEN** nem os logs nem o documento gravado SHALL conter a assinatura da URL (`X-Amz-Signature`)

### Requirement: Resultado único por asset
Rodar o job de novo para o mesmo asset SHALL substituir o resultado anterior.

#### Scenario: Job repetido
- **WHEN** os metadados são salvos duas vezes
- **THEN** SHALL existir um único registro, com o resultado mais recente
