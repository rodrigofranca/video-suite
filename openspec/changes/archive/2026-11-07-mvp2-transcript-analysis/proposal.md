# Proposal

## Why

Segunda entrega do Milestone 2 (AI metadata). Com a transcrição pronta, um modelo de linguagem pode descrever o vídeo: um resumo curto, tags de assunto e capítulos clicáveis. É o que torna uma biblioteca de vídeos navegável e pesquisável. Decisões confirmadas em 2026-11-07: as chamadas a modelos de linguagem passam por um gateway LiteLLM, que no futuro roda num VPS e atende vários projetos por rede privada; o primeiro provedor atrás dele é o OpenCode; o VPS fica para a próxima entrega.

## What Changes

- Job novo no fim do pipeline: `analyze-transcript`, executado pelo worker de IA.
- Gateway LiteLLM no Compose (`litellm`, porta 4000 só local), com o modelo `ultramedia-text` apontando para o OpenCode. A imagem é fixada por digest e tem a assinatura conferida.
- Worker de IA: cliente da API OpenAI só com a biblioteca padrão, com novas tentativas para falhas passageiras; prompt com a transcrição em trechos com horário; resposta JSON validada e limpa.
- Migração `0010_create_analyses`; `GET /assets/{id}/analysis`.
- Dashboard: seção **Summary** com resumo, tags e capítulos que levam o player ao ponto.
- Configuração: `LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL`, `LITELLM_MASTER_KEY`, `OPENCODE_API_KEY`, `OPENCODE_API_BASE`; `make llm-check`.

## Capabilities

### New Capabilities

- `transcript-analysis`: resumo, tags e capítulos escritos por um modelo de linguagem a partir da transcrição.
- `llm-gateway`: o gateway LiteLLM e como os serviços falam com modelos de linguagem.

### Modified Capabilities

- `job-contracts`: tipo `analyze-transcript`, roteado para `jobs:ai`.
- `processing-pipeline`: seis passos.
- `asset-upload`: o upload cria os seis jobs.
- `asset-api`: rota da análise.
- `dashboard`: seção de resumo.
- `media-asset-model`: tabela `analyses`.
- `ai-worker`: verificação do gateway na inicialização.
- `demo-stack`: serviço `litellm`.
- `monorepo-structure`: `make up` sobe o gateway; target `llm-check`.

## Impact

- Código: `apps/ai-worker` (`llm.py`, `analyze.py`), `apps/api`, `apps/dashboard`, `packages/job-contracts`, `packages/media-core`.
- Dependências: nenhuma nova no Python; imagem `ghcr.io/berriai/litellm:v1.108.2` fixada por digest.
- Rede: o gateway chama `opencode.ai`; o worker de IA só chama o gateway.
- Custo: cada vídeo com fala faz uma chamada ao modelo (duas, se a primeira resposta vier inválida).
- Docs: ADRs 0018 e 0019.
