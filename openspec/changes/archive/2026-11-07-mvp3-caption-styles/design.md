## Decisions

**`box` em duas camadas:** o ASS não tem fundo por palavra. A camada 0 desenha a frase em branco; a camada 1 repete o texto num estilo de caixa opaca (BorderStyle 3), com todas as palavras transparentes menos a que está sendo dita. Como o texto é o mesmo, as posições coincidem e a caixa cai exatamente atrás da palavra.

**`karaoke` com a tag do próprio ASS:** um evento por frase, `{\kf<centésimos>}` antes de cada palavra, com a duração até a próxima começar. A cor secundária (branca) vai sendo trocada pela primária (amarela) da esquerda para a direita.

**`one-word` sem frases:** um evento por palavra, até a próxima começar (ou 0,4 s depois do fim), em maiúsculas, com `\t` fazendo a palavra encolher de 115% para 100% em 120 ms. Palavras com mais de 10 letras têm a fonte reduzida na proporção, para caber na largura.

**Nomes:** `highlight` substitui `words`, que só existia na branch ainda não mesclada; não há dados a migrar.
