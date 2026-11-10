# Tasks

- [x] 1 Migração 0019: `outbox_events`, índices e trigger de aviso
- [x] 2 Contratos: nomes dos eventos e do canal, no arquivo compartilhado
- [x] 3 Store: eventos gravados com os jobs e com o resultado; tratamento com `SKIP LOCKED`, savepoint, espera crescente, `LISTEN` e limpeza
- [x] 4 Relay e handlers (`Publish`, `SettleAsset`, `QueueNext`); testes com o banco: pipeline inteiro, HLS com falha, Redis fora do ar, três relays ao mesmo tempo, aviso e limpeza
- [x] 5 API: casos de uso sem a porta `Queue`; relay no processo; testes de integração lendo a outbox
- [x] 6 Worker de mídia: runner sem encadeamento; relay no processo; testes
- [x] 7 Worker de IA: `finish_job` grava o evento na transação; runner sem encadeamento; testes
- [x] 8 ADR 0034; ADR 0009 e 0033; README, CONTEXT e docs/README
- [x] 9 Pilha Docker: pipeline inteiro pelos relays, e upload com o Redis parado, que termina quando ele volta
