# llm-gateway Specification

## Purpose
Define o gateway de modelos de linguagem (LiteLLM) e como os serviços da plataforma falam com ele.

## Requirements

### Requirement: Serviços falam só com o gateway
Os serviços SHALL chamar modelos de linguagem só pelo gateway, com a API de chat completions da OpenAI e um nome de modelo definido em `infra/litellm/config.yaml` (`ultramedia-text`). O provedor, o ID do modelo e a chave do provedor SHALL existir só nesse arquivo e no ambiente do gateway.

#### Scenario: Trocar de modelo
- **WHEN** o modelo de `ultramedia-text` muda no arquivo de configuração e o gateway reinicia
- **THEN** o worker de IA SHALL usar o novo modelo sem mudança de código ou de variável

#### Scenario: Chave do provedor
- **WHEN** o ambiente do worker de IA é inspecionado
- **THEN** ele SHALL ter só a chave do gateway (`LLM_API_KEY`), nunca `OPENCODE_API_KEY`

### Requirement: OpenCode como primeiro provedor
`ultramedia-text` SHALL apontar para um modelo do OpenCode Zen (padrão: `claude-haiku-5-5`) pelo endpoint compatível com a OpenAI, com a URL em `OPENCODE_API_BASE` (padrão `https://opencode.ai/inference/openai/v1`) e a chave em `OPENCODE_API_KEY`.

#### Scenario: Teste da chave
- **WHEN** o desenvolvedor executa `make llm-check` com uma chave válida
- **THEN** o comando SHALL imprimir a resposta do modelo

### Requirement: Imagem fixada e conferida
O serviço `litellm` SHALL usar a imagem fixada por digest, cuja assinatura foi conferida com a chave cosign publicada pelo projeto, nunca uma tag flutuante. SHALL usar a tabela de preços que vem na imagem (`LITELLM_LOCAL_MODEL_COST_MAP`) e telemetria desligada, e SHALL publicar a porta 4000 só em `127.0.0.1`.

#### Scenario: Gateway no ar
- **WHEN** `make up` termina
- **THEN** `http://localhost:4000/health/liveliness` SHALL responder e `GET /v1/models` com a chave mestra SHALL listar `ultramedia-text`

#### Scenario: Chave errada
- **WHEN** uma chamada usa uma chave que o gateway não conhece
- **THEN** o gateway SHALL recusá-la sem chamar o provedor
