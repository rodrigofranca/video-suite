# jump-cuts Specification

## Purpose
Define os cortes secos de uma exportação: o que fica, como é cortado e como o pedido chega à exportação (ADR 0040).

## Requirements

### Requirement: O que fica
Com cortes secos, a exportação SHALL manter cada palavra do corte com 0,10 s antes e 0,15 s depois, juntar duas palavras quando sobram menos de 0,30 s entre elas e cortar o resto. Os vícios de fala ("uh", "um", "hmm", "ãh", "hã", "äh", "euh", com as letras repetidas contando uma vez) SHALL sair; palavras com sentido em algum idioma ("er", "em", "é", "like", "tipo") SHALL ficar. O plano SHALL usar as palavras da transcrição original, em qualquer idioma de legenda.

#### Scenario: Duas pausas longas
- **WHEN** um corte de 30 s tem pausas de 1,4 s e 2,1 s entre as frases
- **THEN** o arquivo SHALL durar cerca de 27 s, e o log SHALL dizer `jump cuts: keeping 27.0 s of 30.0 s in 3 parts`

### Requirement: Como corta
Os trechos mantidos SHALL ser recortados da mesma imagem e do mesmo som e juntos por `concat`, com o som de cada trecho entrando e saindo em 15 ms. O estágio SHALL vir depois da legenda, que é cortada junto com a imagem, e antes das vinhetas, que entram inteiras. Imagem e som SHALL ter a mesma duração, com tolerância de um quadro.

#### Scenario: Sem desvio
- **WHEN** três trechos de um vídeo que muda de cor a cada segundo são mantidos
- **THEN** cada cor SHALL aparecer na ordem, e o som SHALL durar o mesmo que a imagem

### Requirement: Pedido e preset
A exportação SHALL aceitar `jumpCuts`, e o preset SHALL ter `jumpCuts`, que vale quando o pedido não diz nada. Sem o tempo das palavras na transcrição, a API SHALL responder 409 pedindo para transcrever de novo. Um corte sem fala SHALL ser exportado inteiro, com um aviso no log. O arquivo SHALL guardar `jumpCuts`, `keptSeconds` e `fillers`, e a lista de exportações SHALL trazer `jumpCuts` e a nova duração.

#### Scenario: Transcrição antiga
- **WHEN** o pedido traz `jumpCuts: true` e a transcrição não tem o tempo das palavras
- **THEN** a API SHALL responder 409 com "run the transcription again"
