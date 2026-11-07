# clip-export Specification

## Purpose
Define o job `export-clip`, que corta um trecho sugerido do vídeo em MP4, no formato pedido e com legendas opcionais.

## Requirements

### Requirement: Exportação em três formatos
O job SHALL ler início e fim do corte no banco quando roda e gerar um MP4 H.264 + AAC com `faststart` em `assets/{id}/clips/{clipId}/{formato}.mp4`: `original` no formato do vídeo com até 1920 px no lado maior; `vertical` com o centro recortado em 9:16, 1080x1920; `vertical-blur` com o quadro inteiro sobre uma cópia desfocada, 1080x1920. SHALL registrar um artefato `clip` com o corte, o formato, se tem legendas, o idioma, as dimensões e o tamanho, e SHALL reportar progresso.

#### Scenario: Vertical com desfoque
- **WHEN** um corte de 4,5 s de um vídeo 640x360 é exportado em `vertical-blur`
- **THEN** o arquivo SHALL ter 1080x1920, áudio AAC e cerca de 4,5 s

#### Scenario: Corte de outra sugestão
- **WHEN** o corte não existe mais porque os cortes foram sugeridos de novo
- **THEN** o job SHALL falhar dizendo que os cortes provavelmente foram sugeridos de novo

### Requirement: Legendas queimadas
Com `captions`, o job SHALL queimar a fala do intervalo do corte, com os tempos a partir de 0, em fonte DejaVu Sans em negrito com contorno, no estilo `caption_style`: `words` (padrão) ou `lines`, que usa as legendas WebVTT do asset, divididas em pedaços de até 32 caracteres no vertical. Sem palavras na transcrição, `words` SHALL virar `lines`, com o motivo no log; sem legendas, o corte SHALL sair sem elas, também com o motivo no log. O artefato SHALL registrar o estilo usado em `captionStyle`.

#### Scenario: Legenda visível
- **WHEN** o mesmo corte é exportado com e sem legendas
- **THEN** a parte de baixo do quadro SHALL ser diferente entre os dois arquivos

#### Scenario: Transcrição sem palavras
- **WHEN** a transcrição foi feita antes de as palavras serem guardadas
- **THEN** o corte SHALL sair com legendas em linhas e `captionStyle: "lines"`, e o log SHALL dizer que basta transcrever de novo

### Requirement: Legendas palavra a palavra
No estilo `words`, o job SHALL usar as palavras da transcrição em frases curtas (até 4 palavras e 22 caracteres no vertical, 7 e 40 no original), quebrando em fim de frase e em pausas de mais de 0,8 s, e SHALL destacar em amarelo a palavra que está sendo dita.

#### Scenario: Palavra dita destacada
- **WHEN** um corte de um vídeo cinza é exportado em `words` e em `lines`
- **THEN** o quadro do momento de uma palavra SHALL ter pixels amarelos na legenda em `words` e nenhum em `lines`

### Requirement: Entrada validada
O job SHALL recusar formato desconhecido, `clip_id` que não seja UUID, `output_key` diferente da chave do corte e corte de outro asset.

#### Scenario: Chave de saída de outro asset
- **WHEN** `output_key` aponta para outro asset
- **THEN** o job SHALL falhar sem gravar nada
