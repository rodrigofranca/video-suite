# event-outbox Specification

## Purpose
Define a outbox de eventos de domínio: gravados na mesma transação da mudança e tratados por um relay que manda os jobs ao Redis e move o pipeline (ADR 0034).

## Requirements

### Requirement: Eventos gravados com a mudança
Uma mudança que leva a mais trabalho SHALL gravar um evento de domínio em `outbox_events` na mesma transação. Criar um job que pode rodar SHALL gravar `job.queued`. Salvar o resultado de um job, nos dois workers, SHALL gravar `job.finished`. Gravar um evento SHALL avisar os relays pelo canal `outbox_events` quando a transação termina.

#### Scenario: Upload
- **WHEN** o upload termina
- **THEN** SHALL existir um único `job.queued`, o do primeiro job do pipeline, gravado na transação que cria os jobs

#### Scenario: Job termina no worker de IA
- **WHEN** o worker de IA salva o resultado de um job, com sucesso ou falha
- **THEN** SHALL existir um `job.finished` desse job, gravado na mesma transação

### Requirement: Relay
A API e o worker de mídia SHALL rodar um relay que trata os eventos não tratados, do mais antigo ao mais novo, cada um na transação que o marca como tratado. Cada evento SHALL ser pego por um relay de cada vez. O relay SHALL acordar com o aviso do canal e olhar a tabela pelo menos a cada 5 segundos. Um evento de tipo desconhecido SHALL ser marcado como tratado, com um aviso no log.

#### Scenario: Vários relays
- **WHEN** três relays tratam os mesmos eventos ao mesmo tempo
- **THEN** cada job SHALL ser mandado ao Redis uma vez

### Requirement: Reações aos eventos
`job.queued` SHALL mandar a mensagem do job ao stream de `jobcontracts.StreamFor`, se o job ainda está `pending`. `job.finished` SHALL deixar o vídeo `ready` ou `failed` quando o job é o `transcode-hls`, e SHALL pôr na fila o próximo job `pending` do vídeo, pelo `Pipeline` ou pelo `After`, escrevendo qual no log do job que terminou.

#### Scenario: Job que já rodou
- **WHEN** o `job.queued` de um job que já está `running` é tratado
- **THEN** a mensagem SHALL não ser mandada de novo

### Requirement: Tentativas
Quando uma reação falha, o que ela mudou SHALL ser desfeito, o evento SHALL guardar o erro e o número de tentativas, e SHALL esperar antes de tentar de novo: 1 s, 2 s, 4 s e assim por diante, até 5 minutos.

#### Scenario: Redis fora do ar
- **WHEN** o Redis está parado durante um upload
- **THEN** a API SHALL responder 201, os jobs SHALL continuar `pending`, o evento SHALL guardar o erro, e o pipeline SHALL rodar quando o Redis voltar, sem ação manual

### Requirement: Histórico de eventos
Eventos tratados SHALL ficar guardados por 7 dias e depois ser apagados. Os eventos de um vídeo SHALL ser apagados com ele.

#### Scenario: Limpeza
- **WHEN** um evento foi tratado há mais de 7 dias
- **THEN** o relay SHALL apagá-lo
