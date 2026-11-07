## Context

Segunda entrega do Milestone 2. Decisões de arquitetura nas ADRs 0018 (gateway LiteLLM, OpenCode como primeiro provedor) e 0019 (uma análise por asset, depois da transcrição).

## Decisions

**O worker só conhece o gateway:** o worker de IA chama `LLM_BASE_URL` com a API da OpenAI e o nome `ultramedia-text`. Provedor, ID do modelo e chave ficam em `infra/litellm/config.yaml` e no ambiente do gateway. Trocar de modelo ou de provedor é mudar esse arquivo; nenhum serviço muda.

**Uma chamada para tudo:** resumo, tags e capítulos saem de uma resposta JSON só. É mais barato e mais coerente do que três chamadas. A transcrição vai em trechos de cerca de 20 segundos, cada um com `[MM:SS]`, o que reduz tokens em vídeos longos e limita os pontos onde um capítulo pode começar.

**A resposta é conferida, não confiada:** o JSON é extraído mesmo com cercas de código, blocos de raciocínio ou frases em volta. Se não der para usar, o worker pede de novo uma vez, dizendo o erro. Tags são limpas e sem repetição. Capítulos são encaixados no início de trecho mais próximo, ordenados, com o primeiro em 0, pelo menos 10 s entre eles e no mínimo dois. Assim um capítulo nunca leva o player a um ponto sem fala.

**A transcrição é dado, não instrução:** o prompt de sistema diz para ignorar pedidos dentro da transcrição. O texto gerado é mostrado como texto no dashboard, nunca como HTML.

**Novas tentativas no cliente:** 408, 409, 429 e 5xx são tentados até três vezes, com espera crescente ou o `Retry-After` (até 30 s). Erros de autenticação não são repetidos e ganham uma dica de qual chave conferir. Mensagens de erro passam por um filtro que esconde qualquer coisa parecida com chave.

**Modelo que respondeu:** o LiteLLM devolve o apelido no corpo e o modelo real no cabeçalho `x-litellm-model-name`; o worker grava o modelo real.

**Sem dependência nova:** o cliente usa `urllib`. Uma requisição POST por job não justifica uma biblioteca HTTP.

**Imagem do LiteLLM:** a versão 1.107.0 (21/10/2026) foi publicada com um ladrão de credenciais. A imagem é fixada pelo digest da 1.108.2 (LTS), cuja assinatura foi conferida com a chave cosign nova do projeto. `LITELLM_LOCAL_MODEL_COST_MAP` evita baixar a tabela de preços da branch `main` do GitHub a cada início, e a telemetria fica desligada.

## Risks / Trade-offs

- **Qualidade depende do modelo:** o padrão é `deepseek-v4-flash` (barato e rápido); modelos maiores escrevem melhor e custam mais.
- **Vídeos muito longos:** acima de `LLM_MAX_INPUT_CHARS` (200 mil caracteres, cerca de 4 horas de fala) o job falha com mensagem clara, em vez de truncar.
- **Gateway sem banco:** localmente só existe a chave mestra. No VPS, cada projeto deve ter sua chave virtual, o que exige o banco do LiteLLM (próxima entrega).
- **Reexecutar a transcrição não refaz a análise:** como os demais jobs, cada um roda de novo separadamente.
