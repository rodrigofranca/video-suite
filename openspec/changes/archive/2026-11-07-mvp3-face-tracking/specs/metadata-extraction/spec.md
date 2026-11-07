# Spec Delta

## MODIFIED Requirements

### Requirement: Metadados extraídos com ffprobe
O job SHALL rodar ffprobe no original e gravar em `technical_metadata` um documento `{"summary", "ffprobe"}`. O resumo SHALL ter duração, largura, altura, codecs de vídeo e áudio, bitrate, frame rate, container e tamanho. Largura e altura SHALL ser as exibidas: com rotação de exibição de 90° ou 270°, como em vídeos de celular, SHALL vir trocadas em relação às gravadas.

#### Scenario: Vídeo de 8 segundos em 640x360
- **WHEN** o job roda num MP4 H.264 + AAC de 8 s em 640x360
- **THEN** o resumo SHALL trazer duração de cerca de 8 s, `640x360`, `h264` e `aac`

#### Scenario: Arquivo sem vídeo
- **WHEN** o arquivo não tem stream de vídeo
- **THEN** o job SHALL falhar com `the file has no video stream`

#### Scenario: Vídeo de celular em pé
- **WHEN** o vídeo é gravado em 640x360 com rotação de exibição de 90°
- **THEN** o resumo SHALL trazer `360x640`
