# Painel em componentes, com o estado fora da tela

O painel cresceu feature a feature em três arquivos grandes:

| Arquivo | Linhas | O que juntava |
| --- | --- | --- |
| Página do vídeo | 560 | polling, carregamento de transcrição, resumo e cortes, ações e a tela |
| Componente de cortes | 568 | as escolhas de **todos** os cortes, em mapas indexados pelo ID do corte |
| Página de presets | 681 | o formulário de preset, a prévia da marca d'água e duas bibliotecas quase iguais |

Ali estavam também regras que ninguém testava: o aviso antes de sugerir cortes de novo, o enquadramento por rosto, o que um preset faz. O painel não tinha nenhum teste além da checagem de tipos.

**Decisão: componentes pequenos, estado e regras em módulos, e testes unitários** (passo 5 do ADR 0033).

**Regras em funções puras** (`.ts`), testadas sem navegador:

- `jobs.ts`: jobs ativos, o mais recente de um tipo, o texto da codificação, o aviso antes de sugerir cortes de novo;
- `clips.ts`: o enquadramento por rosto, a descrição de um arquivo exportado, o que um preset acrescenta;
- `presets.ts`: o preset em branco, a descrição de vinhetas e marca d'água, a posição da prévia (a mesma conta da exportação);
- `library.ts`: os tipos de arquivo da biblioteca, como na API (ADR 0033): o que cada um aceita, como é nomeado, mostrado, enviado e apagado.

**Estado em classes com runes** (`.svelte.ts`):

- `AssetPage`: o estado da página do vídeo. Faz o polling e carrega transcrição, resumo e cortes quando os jobs terminam. Expõe as ações: rodar de novo, exportar, editar, apagar.
- `JobResult`: o padrão que se repetia três vezes, "carregar quando a execução mais recente de um job terminar". Ele não recarrega só porque o polling rodou de novo.
- `ExportChoices`: as escolhas de exportação de **um** corte. Guarda a legenda, a cor e o caso próprios de cada estilo, aplica um preset e monta o pedido. Cada cartão de corte tem a sua, em vez de mapas por ID.

**Componentes:**

| Componente | Papel |
| --- | --- |
| `JobSection` | resumo, cortes e transcrição: o conteúdo, ou o que o job está fazendo |
| `VideoPlaceholder`, `AssetDetails`, `DeleteVideo` | partes da página do vídeo |
| `ClipCard`, `CaptionLook`, `ExportFormat` | um corte, a aparência da legenda, um formato de exportação |
| `Library` | upload e lista de **qualquer** tipo de arquivo da biblioteca; vinhetas e marcas d'água usam o mesmo |
| `PresetForm` | o formulário de preset, com a prévia da marca d'água |

As páginas ficam com o layout: a do vídeo foi de 560 para 177 linhas, a de presets de 681 para 180, e a lista de cortes de 568 para 34.

**Testes:** o painel ganhou Vitest (`npm test`, também em `make test-dashboard` e `make test`), com testes das regras e das classes de estado. Os `data-testid` e as classes que os testes de navegador usam continuam iguais.

**O que não muda:** o que a tela mostra e faz. As escolhas de um corte continuam valendo até a página recarregar, como antes.

**Para o futuro:** um tipo novo de arquivo na biblioteca é um `LibraryKind` a mais; uma seção nova da página do vídeo, que dependa de um job, é um `JobResult` e um `JobSection`.
