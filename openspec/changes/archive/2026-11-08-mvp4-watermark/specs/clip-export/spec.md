# Spec Delta

## ADDED Requirements

### Requirement: Marca d'água na exportação
Com `watermark`, o job SHALL recusar chaves fora de `library/watermarks/`, cantos, tamanhos e opacidades fora dos limites, e imagens que não abrem. SHALL pôr a imagem sobre o corte, depois do enquadramento e abaixo da legenda, no canto pedido, a 4% do lado menor da borda, com a largura pedida do lado menor e a opacidade pedida, sem cobrir as vinhetas. O artefato SHALL registrar `watermark`.

#### Scenario: Canto e opacidade
- **WHEN** um corte 640x360 tem uma imagem branca no canto de cima à direita, com 25% e opacidade 100
- **THEN** o canto de cima à direita SHALL ficar branco, numa largura de 90 px, e o de cima à esquerda SHALL ficar como estava

#### Scenario: Com abertura
- **WHEN** o corte tem abertura e marca d'água
- **THEN** a abertura SHALL aparecer sem a marca
