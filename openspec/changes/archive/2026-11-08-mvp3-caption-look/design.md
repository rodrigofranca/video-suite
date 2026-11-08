## Context

Decisões na ADR 0025, sobre os estilos da ADR 0021.

## Decisions

**Opções separadas do estilo:** o estilo diz como as palavras aparecem; posição, cor e caixa são ajustes por cima, válidos para todos. No ASS, posição é o alinhamento (2, 5, 8) e a margem vertical; cor é a cor de destaque de cada estilo; maiúsculas é o texto transformado antes de escapar.

**Padrões por estilo:** sem escolha, cada estilo mantém sua aparência: amarelo no destaque e no karaokê, violeta na caixa, maiúsculas só no `one-word`. `caption_caps` é ponteiro no contrato para distinguir "não escolhido" de "desligado".

**Paleta fechada:** cinco cores testadas, em vez de cor livre, para não gerar legendas ilegíveis. O texto da caixa escolhe preto ou branco pela luminância.

**Em cima, abaixo das barras dos apps:** 14% do topo no vertical, 6% no horizontal.

**O que foi usado fica no artefato:** posição, cor (vazia em estilos sem destaque) e maiúsculas, inclusive quando o estilo cai para `lines`.

## Risks / Trade-offs

- **Legenda no meio cobre o rosto** em vídeos de uma pessoa; a escolha fica com quem exporta.
- **Maiúsculas ocupam mais largura:** as frases continuam com os mesmos limites de caracteres, e o ASS quebra a linha se precisar.
