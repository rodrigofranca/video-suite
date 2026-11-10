# ai-usage Specification

## Purpose
Define o registro do uso do modelo de linguagem, chamada por chamada, e os totais por vídeo e por período.

## Requirements

### Requirement: Uma linha por chamada
Cada resposta do modelo a um job SHALL gravar em `llm_usage` o vídeo, o job, o tipo do job, o modelo que respondeu, os tokens de entrada e de saída e o custo em dólares do cabeçalho `x-litellm-response-cost`. Uma resposta que não serviu SHALL contar também. Sem cabeçalho, com valor inválido ou com custo zero numa chamada que usou tokens, o custo SHALL ficar vazio. Uma falha ao gravar SHALL não falhar o job.

#### Scenario: Resposta corrigida
- **WHEN** a análise recebe uma resposta sem JSON e depois uma boa
- **THEN** SHALL haver duas linhas para o job

### Requirement: Linhas que sobrevivem
Apagar um vídeo SHALL manter as linhas do seu uso, sem o vídeo e sem o job. A migração SHALL criar, a partir dos logs dos jobs anteriores com "model X, N tokens in, M out", uma linha por log, sem custo e marcada `from_log`.

#### Scenario: Vídeo apagado
- **WHEN** um vídeo com uso registrado é apagado
- **THEN** os totais do período SHALL continuar com o uso dele, agrupado como vídeo apagado

### Requirement: Totais na API
`GET /assets/{id}` SHALL trazer `aiUsage` com chamadas, tokens de entrada e saída, custo, chamadas sem preço e o total por tipo de job. `GET /usage?days=` SHALL aceitar 1, 7, 30, 90 ou 0 (tudo), responder 400 para outro valor e trazer o total do período e os grupos por dia (UTC), modelo, tipo de job e vídeo (os 20 que mais gastaram).

#### Scenario: Período inválido
- **WHEN** o pedido é `GET /usage?days=5`
- **THEN** a API SHALL responder 400
