# Design

## Quem grava e quem age

| Evento | Gravado por | Tratado por |
| --- | --- | --- |
| `job.queued` | store: `CompleteUpload` (primeiro job), `CreateJobIfIdle`, `EditClipIfIdle` (rastreio) e o próprio relay (próximo job) | `Publish`: `XADD` no stream do job, se ele ainda está `pending` |
| `job.finished` | `FinishJob` do store Go e do store Python | `SettleAsset` (HLS → `ready` ou `failed`) e `QueueNext` (próximo `pending`, pelo `Pipeline` ou pelo `After`; uma linha no log do job) |

## O relay

- **Pega um evento por transação:** `SELECT … WHERE handled_at IS NULL AND next_attempt_at <= NOW() ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`. Vários relays dividem o trabalho sem pegar o mesmo evento.
- **Handlers num savepoint:** se um falha, o que ele mudou é desfeito. O evento ganha `attempts + 1`, `last_error` e `next_attempt_at` = agora + 1 s, 2 s, 4 s… até 5 min.
- **Acorda:** `LISTEN outbox_events` numa conexão própria, que sai do pool. O trigger dá `pg_notify` quando um evento é gravado, e o relay também olha a cada 5 s.
- **Limpa:** apaga uma vez por hora os eventos tratados há mais de 7 dias.
- **Testes:** um relay pode se limitar a alguns vídeos (`Assets`), para não mexer nos eventos de outros testes que rodam ao mesmo tempo.

## Entrega

Pelo menos uma vez. O `Publish` não manda de novo um job que já saiu de `pending`, e os workers ignoram um job já terminado.
