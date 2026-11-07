# Tasks

## 1. Contratos e dados

- [x] 1.1 Tipo `analyze-transcript`, entrada `AnalyzeTranscriptInput`, `Pipeline` com seis passos, rota `jobs:ai`
- [x] 1.2 Arquivo de contratos regenerado; testes Go e Python atualizados
- [x] 1.3 Migração `0010_create_analyses`, `store.SaveAnalysis`/`GetAnalysis`
- [x] 1.4 `GET /assets/{id}/analysis`

## 2. Worker de IA

- [x] 2.1 Cliente do gateway (`llm.py`) com novas tentativas, dicas de erro e chaves escondidas
- [x] 2.2 Job `analyze-transcript` (`analyze.py`): trechos com horário, prompt, JSON validado, pedido de correção, capítulos limpos
- [x] 2.3 Configuração `LLM_*` e verificação do gateway na inicialização
- [x] 2.4 Testes com um servidor falso compatível com a API da OpenAI

## 3. Gateway

- [x] 3.1 `infra/litellm/config.yaml` com `ultramedia-text` → OpenCode
- [x] 3.2 Serviço `litellm` no Compose, imagem fixada por digest e assinatura conferida
- [x] 3.3 `make up` sobe o gateway; `make llm-check`

## 4. Dashboard

- [x] 4.1 Seção **Summary** com resumo, tags e capítulos clicáveis, estados de espera, falha e sem fala

## 5. Entrega

- [x] 5.1 ADRs 0018 e 0019, README, CONTEXT, roteiro de demonstração
- [x] 5.2 Teste de ponta a ponta com Whisper e LiteLLM reais e um provedor simulado
