# Spec Delta

## RENAMED Requirements

- FROM: `### Requirement: Exportação em três formatos`
- TO: `### Requirement: Exportação nos formatos`

## MODIFIED Requirements

### Requirement: Exportação nos formatos
O job SHALL ler início e fim do corte no banco quando roda e gerar um MP4 H.264 + AAC com `faststart` em `assets/{id}/clips/{clipId}/{formato}.mp4`: `original` no formato do vídeo com até 1920 px no lado maior; `vertical` com o centro recortado em 9:16, 1080x1920; `vertical-face`, igual, seguindo o rosto; `vertical-blur` com o quadro inteiro sobre uma cópia desfocada, 1080x1920. SHALL registrar um artefato `clip` com corte, formato, legendas, idioma, dimensões e tamanho, e reportar progresso.

#### Scenario: Vertical com desfoque
- **WHEN** um corte de 4,5 s de um vídeo 640x360 é exportado em `vertical-blur`
- **THEN** o arquivo SHALL ter 1080x1920, áudio AAC e cerca de 4,5 s

#### Scenario: Corte de outra sugestão
- **WHEN** o corte não existe mais porque os cortes foram sugeridos de novo
- **THEN** o job SHALL falhar dizendo que os cortes provavelmente foram sugeridos de novo

## ADDED Requirements

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
