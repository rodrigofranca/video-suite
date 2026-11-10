# Uma tabela de tipos de job e um template para os jobs de LLM

**O problema:** cada tipo de job estava descrito em vários lugares, e nada conferia que eles concordavam:

- o stream, no Go (`aiJobTypes`);
- o nome que as pessoas veem, numa lista do painel;
- quais tipos podem rodar de novo à mão: uma vez na API (`JobInput`) e outra no painel (`job.type !== 'translate-transcript'`);
- os handlers, no `Register` do worker de mídia e num dicionário do worker Python.

Um tipo novo precisava ser lembrado em todos esses lugares. Se faltasse um handler, a falha só aparecia em produção: "unsupported job type".

No worker Python, os três jobs que usam o modelo de linguagem (resumo, cortes, tradução) repetiam os mesmos passos:

1. ler e conferir a entrada;
2. exigir um modelo configurado;
3. ler a transcrição e decidir se há o que pedir;
4. montar o prompt e conferir o tamanho;
5. perguntar, com uma nova tentativa;
6. salvar e registrar modelo e tokens.

**Decisão 1: uma tabela de tipos de job** (`jobcontracts.JobTypes`). Cada entrada tem o nome, o rótulo que as pessoas veem, o worker que roda o job (`media` ou `ai`) e se ele pode rodar de novo à mão. Tudo o que depende do tipo lê a tabela:

| Quem | Usa a tabela para |
| --- | --- |
| `StreamFor` | o stream do worker |
| `testdata/contracts.json` | levar a tabela ao worker Python, que confere os seus tipos |
| API, ao rodar de novo | aceitar só os tipos marcados; uma exportação ou tradução recebe 400 com o motivo |
| API, nas respostas | `label` e `canRerun` em cada job, e `label` no uso de IA por ação |
| Painel | mostrar o `label` e o **Run again** pelo `canRerun`; as listas próprias saíram |
| Testes dos workers | o worker de mídia trata exatamente os tipos `media`, e o de IA exatamente os `ai` (também conferido na partida) |
| Teste da API | há uma entrada montável para cada tipo que pode rodar de novo, e só para eles |

Um tipo novo é uma linha na tabela e o seu handler. Se faltar algo, um teste falha antes do deploy.

**O que fica fora da tabela:** a ordem do pipeline (`Pipeline`) e os encadeamentos (`RerunWith`, `After`) continuam como estão. Um teste confere que todo tipo citado neles está na tabela. As entradas de cada tipo continuam na API, porque dependem do storage, que o pacote de contratos não conhece.

**Decisão 2: um Template Method para os jobs de LLM** (`LLMTask`, em `apps/ai-worker`). O `__call__` da base roda os passos comuns, em ordem, e cada job preenche os seus:

- `input_type`, `system_prompt` e `what` (o que ele pede, para o log);
- `check_input`, quando a entrada tem mais do que o vídeo (a tradução confere o idioma);
- `plan`: os pedidos ao modelo, ou nada a pedir (depois de registrar o motivo);
- `save`: guarda o resultado e diz o que guardou.

**O que a base passa a fazer pelos três jobs:**

- confere o tamanho de **todos** os prompts antes da primeira chamada, então a tradução de um texto longo falha antes de gastar tokens;
- mostra o progresso quando há vários pedidos;
- soma os tokens de todos os pedidos;
- escreve o log no mesmo formato: o que foi salvo, o modelo e os tokens.

Um job novo de LLM, como o texto do post ou a correção da transcrição com glossário do backlog, é uma subclasse com o prompt, o plano e o salvamento.

**O que não muda:** os prompts, a leitura das respostas e o que é salvo. Os testes dos três jobs passam sem alteração.
