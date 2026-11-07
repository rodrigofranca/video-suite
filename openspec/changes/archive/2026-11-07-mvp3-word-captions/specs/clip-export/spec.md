# Spec Delta

## MODIFIED Requirements

### Requirement: Legendas queimadas
Com `captions`, o job SHALL queimar a fala do intervalo do corte, com os tempos a partir de 0, em fonte DejaVu Sans em negrito com contorno, no estilo `caption_style`: `words` (padrão) ou `lines`, que usa as legendas WebVTT do asset, divididas em pedaços de até 32 caracteres no vertical. Sem palavras na transcrição, `words` SHALL virar `lines`, com o motivo no log; sem legendas, o corte SHALL sair sem elas, também com o motivo no log. O artefato SHALL registrar o estilo usado em `captionStyle`.

#### Scenario: Legenda visível
- **WHEN** o mesmo corte é exportado com e sem legendas
- **THEN** a parte de baixo do quadro SHALL ser diferente entre os dois arquivos

#### Scenario: Transcrição sem palavras
- **WHEN** a transcrição foi feita antes de as palavras serem guardadas
- **THEN** o corte SHALL sair com legendas em linhas e `captionStyle: "lines"`, e o log SHALL dizer que basta transcrever de novo

## ADDED Requirements

### Requirement: Legendas palavra a palavra
No estilo `words`, o job SHALL usar as palavras da transcrição em frases curtas (até 4 palavras e 22 caracteres no vertical, 7 e 40 no original), quebrando em fim de frase e em pausas de mais de 0,8 s, e SHALL destacar em amarelo a palavra que está sendo dita.

#### Scenario: Palavra dita destacada
- **WHEN** um corte de um vídeo cinza é exportado em `words` e em `lines`
- **THEN** o quadro do momento de uma palavra SHALL ter pixels amarelos na legenda em `words` e nenhum em `lines`
