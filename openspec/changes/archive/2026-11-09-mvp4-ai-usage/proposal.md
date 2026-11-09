# Proposal

## Why

Pedido em 2026-11-09: guardar no banco quanto de token foi gasto nas ações de IA. Até aqui os tokens só apareciam como texto no log do job.

## What Changes

- A tabela `llm_usage`, com uma linha por chamada ao modelo: vídeo, job, tipo, modelo, tokens de entrada e saída, custo em dólares; a migração lê de volta os logs dos jobs anteriores.
- O worker de IA grava cada resposta do modelo, inclusive as que não serviram, com o custo que o LiteLLM devolve.
- O preço do modelo vai para a configuração do LiteLLM.
- A API devolve o uso de cada vídeo e `GET /usage` com os totais do período.
- O painel mostra o uso nos detalhes do vídeo e na página **AI usage**.

## Capabilities

### New Capabilities

- `ai-usage`: registro e totais do uso do modelo de linguagem.

### Modified Capabilities

- `dashboard`: uso de IA no vídeo e na página nova.

## Impact

- Código: migração 0018, `apps/ai-worker` (`llm.py`, `runner.py`, `answers.py`, `store.py`), `packages/media-core/store/usage.go`, `apps/api` (`usage.go`, `assets.go`), `apps/dashboard` (`/usage`, `UsageChart`), `infra/litellm/config.yaml`.
- Docs: ADR 0032, README, CONTEXT, docs/README e demo.
