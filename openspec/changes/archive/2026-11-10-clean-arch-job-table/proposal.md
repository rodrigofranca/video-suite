# Proposal

## Why

Passo 4 do plano de arquitetura (ADR 0033), pedido em 2026-11-10.

- **Cada tipo de job estava descrito em vários lugares:** a rota no Go, os nomes no painel, os tipos que a API sabe rodar de novo, a regra do painel sobre o que pode rodar de novo, e os handlers de cada worker.
- **Os três jobs que usam o modelo de linguagem repetiam os mesmos passos** no worker Python.

## What Changes

- **Uma tabela de tipos de job** (`jobcontracts.JobTypes`), com nome, rótulo, worker e se pode rodar de novo à mão. Ela alimenta:
  - o stream;
  - o arquivo de contratos compartilhado;
  - a checagem da API ao rodar de novo;
  - os rótulos que a API devolve;
  - os testes de que cada worker trata os seus tipos.
- **A API** devolve `label` e `canRerun` em cada job, e `label` no uso de IA por ação. Pedir para rodar de novo uma exportação ou uma tradução responde 400 com o motivo.
- **O painel** usa o rótulo e o `canRerun` da API, em vez das suas próprias listas.
- **O worker de IA** confere na partida que tem um handler para cada tipo seu. Os jobs que usam o modelo seguem um Template Method (`LLMTask`), e o tamanho de todos os prompts é conferido antes da primeira chamada.

## Capabilities

### Modified Capabilities

- `job-contracts`: a tabela de tipos e o arquivo compartilhado com ela.
- `asset-api`: rótulo e `canRerun` nos jobs.
- `job-rerun`: tipos que só rodam com uma exportação.
- `dashboard`: rótulos e **Run again** pela API.
- `ai-worker`: um handler por tipo.

## Impact

- **Código:**
  - `packages/job-contracts` (`registry.go`, `queue.go`);
  - `apps/api` (`app/videos.go`, `httpapi/assets.go`, `httpapi/usage.go`);
  - `apps/worker` (`jobs/dispatcher.go`);
  - `apps/ai-worker` (`llmtask.py`, `analyze.py`, `clips.py`, `translate.py`, `__main__.py`);
  - `apps/dashboard` (`format.ts`, `JobList.svelte`, páginas do vídeo e de uso).
- **Docs:** ADR 0036, ADR 0033, README, CONTEXT e docs/README.
