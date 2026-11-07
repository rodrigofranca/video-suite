# Spec Delta

## MODIFIED Requirements

### Requirement: Legendas queimadas
Com `captions`, o job SHALL queimar a fala do intervalo do corte, com tempos a partir de 0, em DejaVu Sans negrito com contorno, no estilo `caption_style`: `highlight` (padrão), `box`, `karaoke`, `one-word` ou `lines` (legendas WebVTT, em pedaços de até 32 caracteres no vertical). Sem palavras na transcrição, os outros estilos SHALL virar `lines`; sem legendas, o corte SHALL sair sem elas; nos dois casos, com o motivo no log. O artefato SHALL registrar o estilo usado em `captionStyle`.

#### Scenario: Legenda visível
- **WHEN** o mesmo corte é exportado com e sem legendas
- **THEN** a parte de baixo do quadro SHALL ser diferente entre os dois arquivos

#### Scenario: Transcrição sem palavras
- **WHEN** a transcrição foi feita antes de as palavras serem guardadas
- **THEN** o corte SHALL sair com legendas em linhas e `captionStyle: "lines"`, e o log SHALL dizer que basta transcrever de novo

### Requirement: Legendas palavra a palavra
Os estilos que seguem as palavras SHALL usar as palavras da transcrição: `highlight`, `box` e `karaoke` em frases curtas (até 4 palavras e 22 caracteres no vertical, 7 e 40 no original), quebradas em fim de frase e em pausas de mais de 0,8 s. Em `highlight`, a palavra dita SHALL ficar amarela; em `box`, SHALL ganhar uma caixa violeta atrás; em `karaoke`, cada palavra SHALL ser preenchida de amarelo enquanto é dita. Em `one-word`, SHALL aparecer só a palavra dita, em maiúsculas.

#### Scenario: Palavra dita destacada
- **WHEN** um corte de um vídeo cinza é exportado em `highlight` e em `lines`
- **THEN** o quadro do momento de uma palavra SHALL ter pixels amarelos na legenda em `highlight` e nenhum em `lines`

#### Scenario: Cada estilo com sua cor
- **WHEN** um corte de um vídeo cinza é exportado em cada estilo e se olha o quadro de uma palavra
- **THEN** `highlight` e `karaoke` SHALL ter pixels amarelos, `box` SHALL ter pixels violeta, e `one-word` e `lines` SHALL ter só texto branco
