# Spec Delta

## ADDED Requirements

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
