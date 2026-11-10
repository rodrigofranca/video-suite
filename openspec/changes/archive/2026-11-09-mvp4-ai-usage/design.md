## Context

Decisões na ADR 0032.

## Decisions

**Um ponto de registro:** todo job que usa o modelo passa por `answers.ask`; ele chama `ctx.record_usage` a cada resposta, e o runner grava a linha com o vídeo e o job. Novos jobs de IA são contados sem código a mais.

**Custo do gateway:** o LiteLLM já calcula o custo de cada resposta quando conhece o preço; o preço do modelo fica junto do modelo, na configuração do gateway, e o código não guarda preços. Sem preço, o custo fica vazio e os totais avisam.

**Linhas que sobrevivem:** o vídeo e o job viram `NULL` quando apagados, para os totais do período não mudarem.

**Passado pelos logs:** a migração extrai dos logs as linhas "model X, N tokens in, M out", marcadas como `from_log`.

## Risks / Trade-offs

- **Preço desatualizado:** se o provedor mudar o preço, o custo registrado fica errado até a configuração mudar.
- **Passado aproximado:** um resumo que precisou de correção só tinha a última chamada no log.
