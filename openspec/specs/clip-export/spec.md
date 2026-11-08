# clip-export Specification

## Purpose
Define o job `export-clip`, que corta um trecho sugerido do vídeo em MP4, no formato pedido e com legendas opcionais.

## Requirements

### Requirement: Exportação nos formatos
O job SHALL ler início e fim do corte no banco quando roda e gerar um MP4 H.264 + AAC com `faststart` em `assets/{id}/clips/{clipId}/{formato}.mp4`: `original` no formato do vídeo com até 1920 px no lado maior; `vertical` com o centro recortado em 9:16, 1080x1920; `vertical-face`, igual, seguindo o rosto; `vertical-blur` com o quadro inteiro sobre uma cópia desfocada, 1080x1920. SHALL registrar um artefato `clip` com corte, formato, legendas, idioma, dimensões e tamanho, e reportar progresso.

#### Scenario: Vertical com desfoque
- **WHEN** um corte de 4,5 s de um vídeo 640x360 é exportado em `vertical-blur`
- **THEN** o arquivo SHALL ter 1080x1920, áudio AAC e cerca de 4,5 s

#### Scenario: Corte de outra sugestão
- **WHEN** o corte não existe mais porque os cortes foram sugeridos de novo
- **THEN** o job SHALL falhar dizendo que os cortes provavelmente foram sugeridos de novo

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

### Requirement: Entrada validada
O job SHALL recusar formato desconhecido, `clip_id` que não seja UUID, `output_key` diferente da chave do corte e corte de outro asset.

#### Scenario: Chave de saída de outro asset
- **WHEN** `output_key` aponta para outro asset
- **THEN** o job SHALL falhar sem gravar nada

### Requirement: Recorte seguindo o rosto
Em `vertical-face`, o `x` do recorte 9:16 SHALL seguir os pontos de `face_track` do corte, em retas entre eles, sem sair do quadro, e o artefato SHALL ter `framing: "face"`. Sem caminho (rastreio não rodou, rosto em menos de 30% do corte, quadro já estreito ou tamanho diferente do rastreado), SHALL recortar o centro, com `framing: "center"` e o motivo no log.

#### Scenario: Rosto atravessando o quadro
- **WHEN** um quadrado branco anda da esquerda para a direita e o caminho o acompanha
- **THEN** o quadrado SHALL ficar perto do centro horizontal do vídeo exportado no início e no fim

#### Scenario: Sem rastreio
- **WHEN** o corte não tem `face_track`
- **THEN** o arquivo SHALL ser igual ao recorte central, com `framing: "center"` e um aviso no log

### Requirement: Vídeos girados
O tamanho usado no recorte SHALL ser o exibido, com a rotação de exibição aplicada, como o FFmpeg decodifica.

#### Scenario: Vídeo de celular em pé
- **WHEN** um vídeo gravado em 640x360 com rotação de 90° é exportado em `vertical`
- **THEN** o recorte SHALL partir de um quadro de 360x640

### Requirement: Aparência das legendas
O job SHALL pôr a legenda embaixo (padrão), no meio ou em cima (a 14% do topo no vertical, 6% no horizontal); SHALL usar a cor de destaque escolhida entre amarelo, verde, rosa, azul e violeta, com palavra preta na caixa clara e branca na escura, ou a do estilo sem escolha; e SHALL usar maiúsculas quando pedido (sem escolha, só em `one-word`). O artefato SHALL registrar posição, cor e maiúsculas usadas.

#### Scenario: Verde em cima
- **WHEN** um corte vertical é exportado em `highlight` com posição `top` e cor `green`
- **THEN** o quadro de uma palavra SHALL ter pixels verdes na faixa de cima e nenhum texto no resto, e o artefato SHALL registrar `top`, `green` e sem maiúsculas

#### Scenario: Caixa rosa no meio, em maiúsculas
- **WHEN** um corte vertical é exportado em `box` com `middle`, `pink` e maiúsculas
- **THEN** a caixa rosa SHALL aparecer no meio do quadro, e o artefato SHALL registrar `middle`, `pink` e maiúsculas

#### Scenario: Sem escolhas
- **WHEN** um corte é exportado sem posição, cor nem maiúsculas
- **THEN** cada estilo SHALL sair como antes: embaixo, amarelo ou violeta, maiúsculas só em `one-word`

### Requirement: Legendas em outro idioma
Com `caption_language`, o job SHALL queimar a legenda da tradução da transcrição nesse idioma, em qualquer estilo: as palavras da tradução nos estilos palavra a palavra e a legenda WebVTT traduzida em `lines`. Sem idioma, ou com o idioma da transcrição, SHALL usar a original, mesmo que uma legenda traduzida seja mais nova. O artefato `clip` SHALL registrar `language` e `translatedFrom`.

#### Scenario: Sem tradução
- **WHEN** a exportação pede francês e a transcrição não tem tradução em francês
- **THEN** o job SHALL falhar dizendo que não há tradução para `fr`, sem gerar arquivo

#### Scenario: Idioma fora da lista
- **WHEN** `caption_language` é `ja`
- **THEN** o job SHALL recusar a entrada
