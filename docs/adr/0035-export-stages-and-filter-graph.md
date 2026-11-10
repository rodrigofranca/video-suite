# Exportação em estágios sobre um builder de grafo do FFmpeg

A exportação de um corte cresceu feature a feature: enquadramento, legenda, vinhetas, marca d'água. Tudo saía de uma função, `ClipArgs`, que montava o grafo do FFmpeg colando strings:

- cada parte precisava saber os rótulos das outras (`[base]`, `[cv]`, `[ibg]`);
- a ordem (enquadrar, marca d'água, legenda, vinhetas) estava escondida na sequência dos `if`;
- um rótulo lido duas vezes só aparecia como erro do FFmpeg, na hora de rodar.

As próximas ideias do backlog (borrar rostos, nome na tela, cortar silêncios) cairiam todas na mesma função.

**Decisão: um builder de grafo e uma lista de estágios** (passo 3 do ADR 0033).

**O builder** (`ffmpeg.Graph`) monta o grafo e as entradas:

- `Input` adiciona um arquivo, com as opções só dele (`-ss`, `-t`);
- `Pipe` e `Apply` aplicam filtros a streams e devolvem o stream novo;
- `Split` e `Fork` fazem cópias e filtros com várias saídas (`concat`);
- `Args` devolve as entradas, o `-filter_complex` e os `-map`.

Quem aplica um filtro não escolhe nem conhece rótulos. Filtros aplicados em sequência ficam numa mesma cadeia, o que mantém o grafo curto. Um stream lido duas vezes, que o FFmpeg recusaria, vira erro antes de rodar. O HLS usa o mesmo builder.

**Os estágios** implementam `Stage`: recebem o corte como os estágios anteriores o deixaram (`Media`: imagem, som, tamanho, duração, fps), aplicam seus filtros e o passam adiante. Hoje são quatro:

| Estágio | O que faz |
| --- | --- |
| `Frame` | o formato: original, 9:16 no centro, 9:16 seguindo o rosto, ou o quadro inteiro sobre um desfoque |
| `Watermark` | a imagem num canto, em % do lado menor |
| `Captions` | o arquivo ASS gravado na imagem |
| `Vignettes` | a abertura e o encerramento encaixados e juntados, com som ou silêncio |

**A ordem é decidida por quem monta a exportação**, o job: primeiro a forma da imagem, depois o que é desenhado sobre ela (a legenda por último, para ficar por cima) e as vinhetas no fim, para não receberem nada disso. `ffmpeg.Export` só aplica os estágios na ordem dada e acrescenta a codificação, que continua uma só. O log do job lista os estágios, por exemplo: `crop to 9:16 following the face → watermark → captions → opening`.

**O que não muda:** o arquivo final. Os testes que rodam o FFmpeg conferem os pixels e o som e passam sem alteração: formatos, rosto, legendas, vinhetas e marca d'água. Testes novos conferem o grafo de cada estágio como texto, sem rodar o FFmpeg.

**Para o futuro, cada ideia do backlog é um estágio a mais:**

- **borrar rostos:** um estágio entre o enquadramento e a marca d'água;
- **nome na tela:** um estágio antes da legenda;
- **cortar silêncios:** um estágio antes do enquadramento, que muda a duração. Nesse caso a legenda precisará das novas marcas de tempo.
