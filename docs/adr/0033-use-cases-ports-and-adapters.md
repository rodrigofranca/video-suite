# Casos de uso, portas e adapters na API

O código cresceu feature a feature, e a regra de negócio da API foi ficando dentro dos handlers HTTP. O handler de exportação chegou a 130 linhas: lia o pedido, aplicava o preset, decidia se traduzia, criava os jobs e enfileirava. Isso tinha três custos. Não dava para reaproveitar a regra fora do HTTP, como numa exportação em lote, numa publicação agendada ou numa CLI. Só dava para testar com banco real, sem simular falhas. E cada feature nova empurrava mais regra para o mesmo lugar. Esta decisão define a arquitetura alvo e a forma de chegar lá.

**A arquitetura alvo** é uma Clean Architecture leve, no jeito de Go:

- **Domínio** (`packages/media-core/domain`): entidades, valores e regras que não dependem de nada. Também guarda os erros que as camadas compartilham, `ErrNotFound` e `ErrBusy`, para nenhuma camada depender do pacote de outra para conferir um erro.
- **Casos de uso** (`apps/api/internal/app`): o que a plataforma faz quando alguém pede, separado de como pediu. Um caso de uso confere o pedido, aplica as regras e usa as portas. Responde com um `app.Error`, que diz o tipo do problema (`Invalid`, `NotFound`, `Conflict`, `Unavailable`) e uma mensagem para quem pediu, sem falar de HTTP.
- **Portas**: interfaces pequenas, declaradas por quem as usa: `Assets`, `Clips`, `Speech`, `Brand`, `Jobs`, `Queue`. O `*store.Store` já satisfaz as portas do banco; ligar o caso de uso é passar o mesmo valor várias vezes.
- **Adapters**: o HTTP (`httpapi`) traduz o pedido para o caso de uso e o `app.Error` para um status; o store (PostgreSQL), o storage (S3) e a fila (Redis Streams) ficam atrás das portas.

As dependências apontam para dentro: o HTTP conhece os casos de uso, os casos de uso conhecem o domínio e as portas, e o domínio não conhece ninguém.

**O que não muda:**

- **Leituras ficam no HTTP.** Listar vídeos e mostrar detalhes leem o store direto (um CQRS leve): passar por um caso de uso não acrescentaria regra nenhuma. Só os comandos, que mudam estado, passam pela camada de casos de uso.
- **Sem ORM e sem repositório genérico.** O SQL explícito do store continua; ele é o adapter.
- **Sem interface para tudo.** Uma porta nasce quando um caso de uso precisa dela, com os métodos de que ele precisa.
- **Os workers** já têm um despachante por tipo de job e funções puras testadas; eles entram no plano mais adiante.

**Os testes passam a ter duas camadas.** Cada caso de uso tem testes unitários com fakes das portas, em memória, que rodam sem banco e simulam o que o HTTP não consegue causar: a fila fora do ar, o banco devolvendo erro no meio. Os testes de integração continuam, agora como testes dos adapters e da ligação entre as partes.

**O primeiro caso de uso é a exportação** (`app.ExportClip`). Ele faz tudo o que o handler fazia, na mesma ordem e com as mesmas respostas: confere o vídeo e o corte, junta o preset ao pedido, valida as escolhas, decide a tradução, resolve vinhetas e marca d'água, cria os jobs e enfileira. O handler ficou com 20 linhas.

**A migração é gradual,** um comando por vez, com os testes de integração garantindo que nada muda para quem usa:

1. Os outros comandos da API: editar e apagar corte, apagar vídeo, rodar de novo, upload, presets e biblioteca. As bibliotecas de vinhetas e de marcas d'água viram uma só, genérica.
2. Eventos de domínio e uma outbox transacional no encadeamento de jobs. Hoje o job é gravado no banco e só depois enviado ao Redis; com a outbox, o envio sai da mesma transação.
3. A exportação no worker como um pipeline de estágios (enquadrar, marca d'água, legenda, vinhetas) sobre um builder de grafo do FFmpeg. Isso prepara borrar rostos, nome na tela e cortar silêncios.
4. Um registro único de tipos de job, e um template para os jobs de LLM no worker Python.
5. No painel, as páginas grandes divididas em componentes, e o estado separado da tela.

O login entra depois do passo 1. Cada caso de uso passa a receber quem está pedindo, e a permissão é conferida num lugar só.
