# Spec Delta

## Purpose
Define o job `export-clip`, que corta um trecho sugerido do vídeo em MP4, no formato pedido e com legendas opcionais.

## ADDED Requirements

### Requirement: Exportação em três formatos
O job SHALL ler início e fim do corte no banco quando roda e gerar um MP4 H.264 + AAC com `faststart` em `assets/{id}/clips/{clipId}/{formato}.mp4`: `original` no formato do vídeo com até 1920 px no lado maior; `vertical` com o centro recortado em 9:16, 1080x1920; `vertical-blur` com o quadro inteiro sobre uma cópia desfocada, 1080x1920. SHALL registrar um artefato `clip` com o corte, o formato, se tem legendas, o idioma, as dimensões e o tamanho, e SHALL reportar progresso.

#### Scenario: Vertical com desfoque
- **WHEN** um corte de 4,5 s de um vídeo 640x360 é exportado em `vertical-blur`
- **THEN** o arquivo SHALL ter 1080x1920, áudio AAC e cerca de 4,5 s

#### Scenario: Corte de outra sugestão
- **WHEN** o corte não existe mais porque os cortes foram sugeridos de novo
- **THEN** o job SHALL falhar dizendo que os cortes provavelmente foram sugeridos de novo

### Requirement: Legendas queimadas
Com `captions`, o job SHALL usar as legendas WebVTT do asset no intervalo do corte, com os tempos a partir de 0, divididas em pedaços de até 32 caracteres no vertical, em fonte DejaVu Sans em negrito com contorno, mais altas no vertical. Sem legendas no asset ou no intervalo, SHALL exportar sem elas e registrar o motivo no log.

#### Scenario: Legenda visível
- **WHEN** o mesmo corte é exportado com e sem legendas
- **THEN** a parte de baixo do quadro SHALL ser diferente entre os dois arquivos

### Requirement: Entrada validada
O job SHALL recusar formato desconhecido, `clip_id` que não seja UUID, `output_key` diferente da chave do corte e corte de outro asset.

#### Scenario: Chave de saída de outro asset
- **WHEN** `output_key` aponta para outro asset
- **THEN** o job SHALL falhar sem gravar nada
