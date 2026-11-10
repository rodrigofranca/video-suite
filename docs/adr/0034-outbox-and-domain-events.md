# Outbox transacional e eventos de domínio no encadeamento de jobs

Até aqui, colocar um job na fila eram dois passos separados: gravar a linha no PostgreSQL e depois mandar a mensagem ao Redis (`XADD`). Entre os dois, um processo podia cair ou o Redis podia estar fora do ar. A API, nesse caso, marcava o job como `failed` com "could not enqueue job" e respondia 500. O worker que terminava um job e não conseguia enfileirar o próximo marcava o próximo e o vídeo como `failed`. Uma queda do Redis de poucos segundos derrubava vídeos inteiros, que só voltavam rodando de novo à mão. E a regra de "quando um job termina, o que vem depois" estava escrita duas vezes, no worker em Go e no worker em Python.

**Decisão: uma outbox transacional, com eventos de domínio.** Uma mudança que leva a mais trabalho grava um evento na tabela `outbox_events`, na mesma transação da mudança. Um relay lê os eventos e age sobre eles. Ou a mudança e o evento são gravados juntos, ou nenhum dos dois: o trabalho não se perde entre o banco e a fila.

**Dois eventos, com nomes em `packages/job-contracts`**, porque o worker Python também os grava, e o arquivo de contratos compartilhado os confere:

- **`job.queued`**: um job pendente pode rodar. Quem grava: o upload (o primeiro job do pipeline), rodar de novo, exportar (a tradução, quando vem antes) e editar um corte (o rastreio de rostos). O store grava o evento junto com o job, então os casos de uso da API não falam mais com o Redis, e a porta `Queue` saiu.
- **`job.finished`**: um job terminou, com sucesso ou falha. Quem grava: o `FinishJob` dos dois workers, na mesma transação do resultado.

**O relay** (`packages/media-core/outbox`) entrega cada evento aos handlers do seu tipo, na transação que marca o evento como tratado:

- `job.queued` → `Publish`: manda a mensagem do job ao seu stream. Um job que não está mais `pending` não é mandado de novo.
- `job.finished` → `SettleAsset`: o job HLS decide se o vídeo fica `ready` ou `failed` (ADR 0003).
- `job.finished` → `QueueNext`: o próximo job pendente do vídeo é colocado na fila, pelo `Pipeline` ou pelo `After`, e o log do job que terminou diz qual (ADR 0009).

A regra de encadeamento agora está num lugar só, em Go. O worker Python só lê o seu stream e grava resultados.

**Como o relay trabalha:**

- **Onde roda:** dentro da API e do worker de mídia. Cada evento é pego por um relay de cada vez (`FOR UPDATE SKIP LOCKED`), então rodar vários é seguro e um cobre o outro.
- **Acorda na hora:** um trigger faz `pg_notify` no canal `outbox_events` quando um evento é gravado. O relay escuta o canal e, por garantia, olha a tabela a cada 5 segundos.
- **Falhas:** se um handler falha, o que ele mudou é desfeito (savepoint) e o evento espera para tentar de novo: 1 s, 2 s, 4 s e assim por diante, até 5 minutos. Com o Redis fora do ar, os jobs ficam `pending` e saem quando ele volta, sem ninguém rodar nada à mão.
- **Histórico:** eventos tratados ficam 7 dias, como histórico, e depois são apagados. Os eventos de um vídeo vão embora com ele.

**Entrega pelo menos uma vez.** Se o relay cair depois do `XADD` e antes de marcar o evento, a mensagem pode ir duas vezes. Os workers já ignoram um job que terminou, e o `Publish` não manda de novo um job que já saiu de `pending`.

**Alternativas descartadas:**

- **Tentar de novo dentro do `XADD`** não resolve uma queda do processo entre os dois passos.
- **Change data capture (Debezium) ou filas transacionais** trariam mais um serviço para operar.
- **Um processo `relay` separado** ficaria para quando houver muitos vídeos ao mesmo tempo. Hoje, rodar dentro dos processos que já existem basta.

**Para o futuro:** outros eventos (vídeo pronto, corte exportado) podem alimentar webhooks e a publicação nas redes do backlog, com novos handlers, sem mexer em quem grava o evento.
