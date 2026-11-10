# quote-clips Specification

## Purpose
Define o corte que alguém faz a partir de uma citação na transcrição: a escolha das palavras, o intervalo, o título e o lugar do corte na lista (ADR 0039).

## Requirements

### Requirement: Criar um corte de uma citação
`POST /assets/{id}/clips` com `start` e `end` (segundos) e `title` opcional SHALL criar um corte com `origin` `quote`, sem job e sem nota, e responder 201 com o corte. O corte SHALL durar de 3 s a 3 minutos e terminar dentro do vídeo, com 400 caso contrário, e SHALL esperar o vídeo sem jobs pendentes ou rodando, com 409 enquanto houver. A mesma transação SHALL enfileirar um `track-faces` só para o corte.

#### Scenario: Corte criado
- **WHEN** o pedido é `{"start": 12.0, "end": 15.2}` num vídeo parado
- **THEN** a API SHALL responder 201 com `origin` `quote`, `score` `null` e `rank` 1, e um `track-faces` SHALL ser enfileirado

### Requirement: Título da citação
Sem título, o título SHALL ser o que foi dito no intervalo: as palavras cujo meio cai dentro dele, ou os trechos, numa transcrição sem tempo de palavra. Com mais de 120 caracteres, SHALL ser cortado numa palavra, com "…". Sem fala no intervalo, SHALL ser "Clip at m:ss".

#### Scenario: Sem título
- **WHEN** o intervalo de 12,0 a 15,2 s contém "A reforma foi aprovada."
- **THEN** o título SHALL ser "A reforma foi aprovada."

### Requirement: Ordem e sugestões novas
O corte de citação SHALL ser o primeiro da lista, e os outros SHALL descer uma posição. Sugerir cortes de novo SHALL mantê-lo, deixar de fora as sugestões que repetem mais da metade dele e colocá-lo antes de qualquer corte com nota.

#### Scenario: Sugerir de novo
- **WHEN** há um corte de citação e o modelo sugere outro com nota 10
- **THEN** o corte de citação SHALL ficar com `rank` 1 e a sugestão com `rank` 2

### Requirement: Escolher a citação no painel
A transcrição SHALL ter **✂ Clip a quote**: o primeiro toque marca a primeira palavra, o segundo a última, em qualquer ordem, e um terceiro recomeça. Sem tempo de palavra, SHALL escolher trechos inteiros. O intervalo SHALL ir do começo da primeira palavra ao fim da última, com até 0,15 s antes e 0,25 s depois, sem alcançar a palavra vizinha. Uma barra SHALL mostrar a citação, o intervalo, a duração e o título preenchido com a citação, com **Preview**, **Make the clip** e **Cancel**; **Make the clip** SHALL ficar desligado com menos de 3 s, mais de 3 minutos ou enquanto um job do vídeo roda. O corte criado SHALL aparecer primeiro em **Clips**, com a etiqueta **Quote** no lugar do número e da nota.

#### Scenario: Uma palavra só
- **WHEN** o usuário escolhe uma única palavra curta
- **THEN** a barra SHALL dizer "A clip lasts at least 3 s: pick more words." e **Make the clip** SHALL ficar desligado

### Requirement: Achar na transcrição
A transcrição SHALL ter **Find in the transcript**, que acha o texto sem ligar para acentos nem maiúsculas, marca onde ele está, mostra "n of N" e pula entre os achados com ▲ e ▼.

#### Scenario: Sem acento
- **WHEN** o usuário procura `tributaria`
- **THEN** "Tributária" SHALL aparecer marcada
