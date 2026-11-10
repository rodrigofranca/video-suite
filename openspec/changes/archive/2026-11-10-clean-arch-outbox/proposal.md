# Proposal

## Why

Passo 2 do plano de arquitetura (ADR 0033), pedido em 2026-11-10. Gravar um job e mandá-lo ao Redis eram dois passos separados. Uma queda do Redis, ou do processo entre os dois passos, marcava jobs e vídeos como `failed`, e só rodando de novo à mão eles voltavam. A regra de "o que vem depois" estava escrita duas vezes, em Go e em Python.

## What Changes

- A tabela `outbox_events`, com um trigger que avisa os relays pelo canal `outbox_events`.
- Dois eventos de domínio, com nomes em `packages/job-contracts`:
  - `job.queued`, gravado com o job pelo upload, ao rodar de novo, ao exportar e ao editar um corte;
  - `job.finished`, gravado pelos dois workers junto com o resultado.
- Um relay (`packages/media-core/outbox`) que roda na API e no worker de mídia:
  - manda os jobs ao Redis;
  - deixa o vídeo `ready` ou `failed` pelo HLS;
  - põe o próximo job na fila;
  - tenta de novo com espera crescente.
- Os casos de uso da API e os workers não mandam mais nada ao Redis. A porta `Queue` saiu, e o worker Python não encadeia mais.

## Capabilities

### New Capabilities

- `event-outbox`: eventos de domínio gravados na transação e tratados pelo relay.

### Modified Capabilities

- `processing-pipeline`: o encadeamento e o status do vídeo passam pelo relay; a fila fora do ar não marca nada como `failed`.
- `asset-upload`: o primeiro job vai para a fila pela outbox; sem 500 quando o Redis está fora do ar.
- `job-rerun`: os jobs de `RerunWith` vão para a fila pelo relay.
- `clip-editing`: o rastreio vai para a fila na mesma transação da edição.
- `job-contracts`: nomes dos eventos e do canal no contrato compartilhado.

## Impact

- Código:
  - migração 0019;
  - `packages/job-contracts`;
  - `packages/media-core` (`domain/events.go`, `store/events.go`, `store/store.go`, `outbox/`);
  - `apps/api` (casos de uso, router, `main.go`);
  - `apps/worker` (runner, `main.go`);
  - `apps/ai-worker` (`store.py`, `runner.py`, `contracts.py`, `__main__.py`).
- Docs: ADR 0034, ADR 0009 e 0033 atualizados, README, CONTEXT e docs/README.
