## MODIFIED Requirements

### Requirement: Job que espera outro
O pacote SHALL definir `After`, com o job que pode esperar por um job fora do pipeline: `translate-transcript` é seguido por `export-clip`. Quando um job termina, o relay SHALL pôr na fila o job pendente do tipo seguinte, pelo pipeline ou por `After`. O arquivo de contratos compartilhado SHALL trazer `after`.

#### Scenario: Tradução termina
- **WHEN** um `translate-transcript` termina, com sucesso ou não, e há um `export-clip` pendente do vídeo
- **THEN** o relay SHALL pôr a exportação em `jobs`

## ADDED Requirements

### Requirement: Eventos de domínio no contrato
O pacote SHALL definir os nomes dos eventos `job.queued` e `job.finished` e o canal `outbox_events`. O arquivo de contratos compartilhado SHALL trazê-los em `events`, e o worker de IA SHALL usar os mesmos nomes.

#### Scenario: Nomes conferidos
- **WHEN** os testes do worker de IA leem o arquivo de contratos
- **THEN** os nomes dos eventos e do canal SHALL ser os mesmos do Go
