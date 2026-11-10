## MODIFIED Requirements

### Requirement: Rastreio depois da edição
Quando início ou fim mudam, a mesma transação SHALL limpar `face_track` do corte, criar um `track-faces` `pending` com `clip_id` e pô-lo na fila pela outbox (ADR 0034), para o stream `jobs:ai`.

#### Scenario: Novo intervalo
- **WHEN** o corte passa de 12–47,5 s para 10,25–40 s
- **THEN** `faceCoverage` SHALL ficar nulo e SHALL haver um `track-faces` enfileirado com o `clip_id` do corte
