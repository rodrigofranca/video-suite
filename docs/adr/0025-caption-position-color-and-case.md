# Posição, cor do destaque e maiúsculas das legendas, escolhidas na exportação

Os cinco estilos de legenda (ADR 0021) vinham com posição, cores e caixa fixas. Agora, a cada exportação, também se escolhe onde a legenda fica, a cor do destaque e se o texto sai todo em maiúsculas. Essas opções estavam pedidas no backlog desde os estilos.

**Posição:** embaixo (o padrão de antes), no meio ou em cima. Embaixo, cada estilo mantém a margem que já tinha, alta no vertical para não ficar sob os botões dos apps. No meio, a legenda fica centrada na altura. Em cima, fica a 14% do topo no vertical, abaixo das barras que os apps desenham ali, e a 6% no horizontal. No arquivo ASS, isso é só o alinhamento do estilo (2, 5 ou 8) e a margem vertical, aplicados também aos estilos extras da caixa e do karaokê.

**Cor do destaque:** amarelo, verde, rosa, azul ou violeta, uma paleta pequena e testada, sem cor livre, para não gerar combinações ilegíveis. Vale para a palavra destacada, a caixa e o preenchimento do karaokê; `lines` e `one-word` não têm destaque. Cada estilo continua com a sua cor quando nenhuma é escolhida: amarelo no destaque e no karaokê, violeta na caixa. Na caixa, a palavra fica preta sobre cores claras (amarelo, verde) e branca sobre as escuras, pela luminância da cor.

**Maiúsculas:** liga ou desliga para todos os estilos. Sem escolha, só `one-word` sai em maiúsculas, como antes. No contrato, `caption_caps` é um ponteiro: ausente significa "o padrão do estilo", o que mantém iguais as exportações já pedidas antes desta mudança.

As três opções são campos opcionais de `ExportClipInput` (`caption_position`, `caption_color`, `caption_caps`), validados pela API (400 para valores fora das listas). O worker grava no artefato o que de fato usou, e a API devolve isso em cada exportação. Quando um estilo de palavras cai para `lines` porque a transcrição não tem os tempos das palavras, posição e maiúsculas continuam valendo, e a cor é descartada.

No painel, as opções aparecem abaixo de **Captions** enquanto há legenda: um seletor de posição, bolinhas de cor só para os estilos com destaque, e **CAPS**. A exportação pronta mostra o que mudou em relação ao padrão, por exemplo "Word box, top, green, caps".
