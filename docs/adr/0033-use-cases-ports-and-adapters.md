# Casos de uso, portas e adapters na API

O código cresceu feature a feature, e a regra de negócio da API foi ficando dentro dos handlers HTTP. O handler de exportação chegou a 130 linhas: lia o pedido, aplicava o preset, decidia se traduzia, criava os jobs e enfileirava. Isso tinha três custos. Não dava para reaproveitar a regra fora do HTTP, como numa exportação em lote, numa publicação agendada ou numa CLI. Só dava para testar com banco real, sem simular falhas. E cada feature nova empurrava mais regra para o mesmo lugar. Esta decisão define a arquitetura alvo e a forma de chegar lá.

**A arquitetura alvo** é uma Clean Architecture leve, no jeito de Go:

- **Domínio** (`packages/media-core/domain`): entidades, valores e regras que não dependem de nada. Também guarda os erros que as camadas compartilham (`ErrNotFound`, `ErrBusy`, `ErrNameTaken`, `InUseError`), para nenhuma camada depender do pacote de outra para conferir um erro.
- **Casos de uso** (`apps/api/internal/app`): o que a plataforma faz quando alguém pede, separado de como pediu. Um caso de uso confere o pedido, aplica as regras e usa as portas. Responde com um `app.Error`, que diz o tipo do problema (`Invalid`, `NotFound`, `Conflict`, `Unavailable`, `TooLarge`, `Unsupported`) e uma mensagem para quem pediu, sem falar de HTTP.
- **Portas**: interfaces pequenas, declaradas por quem as usa: `Assets`, `AssetWriter`, `Metadata`, `Clips`, `ClipWriter`, `Speech`, `Brand`, `PresetWriter`, `LibraryWriter`, `Jobs`, `Queue` e `Files` (o storage). O `*store.Store` já satisfaz as portas do banco; ligar o caso de uso é passar o mesmo valor várias vezes.
- **Adapters**: o HTTP (`httpapi`) traduz o pedido para o caso de uso e o `app.Error` para um status; o store (PostgreSQL), o storage (S3) e a fila (Redis Streams) ficam atrás das portas.

As dependências apontam para dentro: o HTTP conhece os casos de uso, os casos de uso conhecem o domínio e as portas, e o domínio não conhece ninguém.

**O que não muda:**

- **Leituras ficam no HTTP.** Listar vídeos e mostrar detalhes leem o store direto (um CQRS leve): passar por um caso de uso não acrescentaria regra nenhuma. Só os comandos, que mudam estado, passam pela camada de casos de uso.
- **Sem ORM e sem repositório genérico.** O SQL explícito do store continua; ele é o adapter.
- **Sem interface para tudo.** Uma porta nasce quando um caso de uso precisa dela, com os métodos de que ele precisa.
- **Os workers** já têm um despachante por tipo de job e funções puras testadas; eles entram no plano mais adiante.

**Os testes passam a ter duas camadas.** Cada caso de uso tem testes unitários com fakes das portas, em memória, que rodam sem banco e simulam o que o HTTP não consegue causar: a fila fora do ar, o banco devolvendo erro no meio. Os testes de integração continuam, agora como testes dos adapters e da ligação entre as partes.

**O primeiro caso de uso é a exportação** (`app.ExportClip`). Ele faz tudo o que o handler fazia, na mesma ordem e com as mesmas respostas: confere o vídeo e o corte, junta o preset ao pedido, valida as escolhas, decide a tradução, resolve vinhetas e marca d'água, cria os jobs e enfileira. O handler ficou com 20 linhas.

**O passo 1 terminou.** Todos os comandos da API são casos de uso:

| Comando | Caso de uso |
| --- | --- |
| Upload de vídeo | `app.UploadVideo` |
| Apagar vídeo | `app.DeleteVideo` |
| Rodar um job de novo | `app.RerunJob` |
| Editar corte | `app.EditClip` |
| Apagar corte | `app.DeleteClip` |
| Exportar corte | `app.ExportClip` |
| Criar e mudar preset | `app.SavePreset` |
| Apagar preset | `app.DeletePreset` |
| Subir e apagar vinheta ou marca d'água | `app.Library` |

A biblioteca é uma só, com uma estratégia por tipo de arquivo (`app.FileKind`). Adicionar e remover funcionam igual para todo tipo: nome, upload, registro, limpeza se algo falha e o 409 enquanto um preset usa o arquivo. O que muda fica no tipo: o que aceita, o limite, a chave no storage e o registro. As vinhetas são MP4 de até 100 MB, lidos em streaming. As marcas d'água são PNG ou JPEG de até 5 MB, lidos inteiros para conferir o tamanho da imagem. Um tipo novo, como uma trilha sonora ou uma fonte, é um tipo a mais, sem handler novo.

O HTTP ficou só com o que é HTTP: ler o multipart e o JSON, o limite do corpo (`http.MaxBytesReader`, que responde 413) e o status de cada tipo de erro (`TooLarge` vira 413 e `Unsupported` vira 415). As regras que estavam repetidas nos handlers foram para um lugar só: nome, nome de arquivo, conferência de MP4, leitor que conta e calcula o hash, e a validação do corte (`domain.Clip.Problem`). Só uma resposta mudou: um pedido com corpo inválido recebe 400 antes de o ID ser conferido, como já acontecia na exportação.

**O passo 2 terminou** (ADR 0034). Os jobs vão para a fila por uma outbox transacional, com os eventos `job.queued` e `job.finished`. Os casos de uso não falam mais com o Redis, e a porta `Queue` saiu. O encadeamento ficou num lugar só, em Go.

**O passo 3 terminou** (ADR 0035). A exportação no worker é uma lista de estágios (enquadramento, marca d'água, legenda, vinhetas) sobre um builder do grafo do FFmpeg. O job escolhe a ordem, e cada estágio só conhece o corte que recebe.

**A migração é gradual,** um comando por vez, com os testes de integração garantindo que nada muda para quem usa:

1. Os outros comandos da API: editar e apagar corte, apagar vídeo, rodar de novo, upload, presets e biblioteca. As bibliotecas de vinhetas e de marcas d'água viram uma só, genérica.
2. Eventos de domínio e uma outbox transacional no encadeamento de jobs. Hoje o job é gravado no banco e só depois enviado ao Redis; com a outbox, o envio sai da mesma transação.
3. A exportação no worker como um pipeline de estágios (enquadrar, marca d'água, legenda, vinhetas) sobre um builder de grafo do FFmpeg. Isso prepara borrar rostos, nome na tela e cortar silêncios.
4. Um registro único de tipos de job, e um template para os jobs de LLM no worker Python.
5. No painel, as páginas grandes divididas em componentes, e o estado separado da tela.

O login entra depois do passo 1. Cada caso de uso passa a receber quem está pedindo, e a permissão é conferida num lugar só.
