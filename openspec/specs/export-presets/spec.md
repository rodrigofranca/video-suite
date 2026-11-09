# export-presets Specification

## Purpose
Define a biblioteca de vinhetas e os presets de exportação: escolhas de legenda e vinhetas de abertura e encerramento guardadas com um nome.

## Requirements

### Requirement: Biblioteca de vinhetas
`POST /vignettes/upload` SHALL aceitar um campo `name` opcional antes do arquivo e um MP4 de até 100 MB no campo `file`, gravá-lo em `library/vignettes/{id}.mp4`, fora das pastas públicas, e responder 201. `GET /vignettes` SHALL listar as vinhetas, da mais nova para a mais antiga, com os presets que usam cada uma. `GET /vignettes/{id}/file` SHALL redirecionar para um link assinado.

#### Scenario: Sem nome
- **WHEN** uma vinheta `Thanks for watching.mp4` é enviada sem `name`
- **THEN** o nome SHALL ser `Thanks for watching`

#### Scenario: Arquivo que não é MP4
- **WHEN** o arquivo não começa com a caixa `ftyp`
- **THEN** a API SHALL responder 415

### Requirement: Vinheta em uso
`DELETE /vignettes/{id}` SHALL apagar a vinheta e o arquivo e responder 204. Enquanto um preset a usa, SHALL responder 409 com o nome dos presets.

#### Scenario: Em uso
- **WHEN** um preset usa a vinheta como abertura
- **THEN** apagá-la SHALL dar 409, e o arquivo SHALL continuar no storage

### Requirement: Presets
`POST /presets` e `PUT /presets/{id}` SHALL aceitar um nome de 1 a 60 caracteres, as escolhas de legenda e as vinhetas `introHorizontalId`, `introVerticalId`, `outroHorizontalId` e `outroVerticalId`. Valores fora das listas de legenda ou vinhetas desconhecidas SHALL dar 400. `GET /presets` SHALL listar por nome, e `DELETE /presets/{id}` SHALL responder 204.

#### Scenario: Vinheta desconhecida
- **WHEN** o preset aponta para uma vinheta que não existe
- **THEN** a API SHALL responder 400

### Requirement: Nome único
O nome de um preset SHALL ser único sem diferenciar maiúsculas; um nome repetido SHALL dar 409.

#### Scenario: Nome repetido
- **WHEN** já existe `My channel` e outro preset é criado como `my channel`
- **THEN** a API SHALL responder 409

### Requirement: Exportação com preset
Com `presetId`, a exportação SHALL copiar para o job as chaves das vinhetas do preset e o nome dele, e usar as escolhas de legenda do preset que o pedido deixa de fora. As que o pedido traz SHALL valer mais. Um preset desconhecido SHALL dar 400.

#### Scenario: Cor trocada no pedido
- **WHEN** o preset tem caixa rosa e o pedido traz `captionColor` `green`
- **THEN** o job SHALL ter o estilo `box`, a cor `green` e as vinhetas do preset

### Requirement: Imagens de marca d'água
`POST /watermarks/upload` SHALL aceitar um campo `name` opcional antes do arquivo e um PNG ou JPEG de até 5 MB e 8 a 4096 px de lado, gravá-lo em `library/watermarks/{id}.png` ou `.jpg`, fora das pastas públicas, e responder 201 com a largura e a altura. Outro formato SHALL dar 415.

#### Scenario: JPEG
- **WHEN** um JPEG é enviado
- **THEN** SHALL ser gravado com a extensão `.jpg`

### Requirement: Biblioteca de marcas d'água
`GET /watermarks` SHALL listar as imagens com os presets que usam cada uma, e `GET /watermarks/{id}/file` SHALL redirecionar para um link assinado. `DELETE /watermarks/{id}` SHALL responder 204, ou 409 enquanto um preset a usa.

#### Scenario: Em uso
- **WHEN** um preset usa a imagem
- **THEN** apagá-la SHALL dar 409

### Requirement: Marca d'água no preset
Um preset SHALL aceitar `watermarkId`, `watermarkPosition` (`top-right`, `top-left`, `bottom-right`, `bottom-left`), `watermarkSize` (5 a 40, porcentagem do lado menor do corte) e `watermarkOpacity` (10 a 100). Os ausentes SHALL valer `top-right`, 15 e 80. Valores fora disso ou uma imagem desconhecida SHALL dar 400. A exportação com o preset SHALL copiar para o job a chave da imagem e as escolhas.

#### Scenario: Só a imagem
- **WHEN** o preset é criado só com `watermarkId`
- **THEN** SHALL ficar no canto de cima à direita, com 15% e opacidade 80
