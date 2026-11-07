# ai-worker Specification

## Purpose
Define o worker Python (`apps/ai-worker`) que executa os jobs de IA, com o mesmo comportamento de fila e ciclo de vida do worker Go.

## Requirements

### Requirement: Fila própria
O worker SHALL consumir o stream `jobs:ai` com o consumer group `ai-workers`, uma mensagem por vez, e SHALL ackar toda mensagem processada, inclusive com falha, mensagem malformada ou tipo desconhecido.

#### Scenario: Mensagens variadas
- **WHEN** chegam jobs com sucesso, com falha, de tipo desconhecido e uma mensagem malformada
- **THEN** nenhuma mensagem SHALL ficar pendente depois do processamento

### Requirement: Ciclo de vida como no worker Go
O worker SHALL marcar o job `running`, gravar os logs do handler em `job_logs`, gravar o progresso no máximo uma vez por segundo, marcar `done` (progresso 100) ou `failed` com a mensagem, aplicar o limite de 30 minutos, ignorar entregas duplicadas de jobs já concluídos e enfileirar o próximo job do pipeline no stream de quem o executa.

#### Scenario: Job com falha
- **WHEN** o handler levanta um erro
- **THEN** o job SHALL ficar `failed` com a mensagem e o log SHALL ter a linha `job failed: <mensagem>`

#### Scenario: Encadeamento
- **WHEN** um job que tem próximo passo termina
- **THEN** o próximo job pendente SHALL ser enfileirado no stream indicado pelos contratos

### Requirement: Retomada e encerramento
Ao receber SIGINT ou SIGTERM, o worker SHALL parar entre trechos de trabalho, deixar o job `running` e a mensagem sem ack, e encerrar com código zero. Ao iniciar, SHALL reprocessar uma vez as mensagens pendentes do próprio nome (`AI_WORKER_NAME`, padrão hostname).

#### Scenario: Encerramento no meio do job
- **WHEN** o worker recebe SIGTERM durante uma transcrição
- **THEN** o job SHALL continuar `running` e rodar de novo no próximo start

### Requirement: Verificações na inicialização
O worker SHALL exigir `REDIS_URL`, `DATABASE_URL` e as variáveis `S3_*` obrigatórias, conferir Redis, PostgreSQL e o bucket, e carregar o modelo Whisper antes de consumir jobs, encerrando com código não-zero e mensagem clara se algo falhar. SHALL também carregar o detector de rostos, só avisando quando não consegue. Com `LLM_BASE_URL`, SHALL conferir se o gateway responde e oferece `LLM_MODEL`, apenas avisando quando não.

#### Scenario: Redis fora do ar
- **WHEN** o Redis não responde na inicialização
- **THEN** o worker SHALL logar `ai worker stopped: could not connect` e encerrar com código 1

#### Scenario: Gateway fora do ar
- **WHEN** o gateway não responde na inicialização
- **THEN** o worker SHALL logar um aviso e continuar consumindo jobs

#### Scenario: Sem gateway configurado
- **WHEN** `LLM_BASE_URL` está vazia
- **THEN** o worker SHALL avisar que os jobs `analyze-transcript` vão falhar e continuar consumindo jobs

#### Scenario: Detector de rostos indisponível
- **WHEN** o detector não pode ser baixado na inicialização
- **THEN** o worker SHALL avisar e continuar consumindo jobs, e o job `track-faces` SHALL tentar carregá-lo de novo

### Requirement: Contratos iguais aos do Go
Os tipos de job, a ordem do pipeline, os streams, os consumer groups, os campos da mensagem e as entradas do worker SHALL coincidir com `packages/job-contracts/testdata/contracts.json`.

#### Scenario: Contrato alterado só no Go
- **WHEN** a ordem do pipeline muda no Go e o arquivo de contratos é regenerado
- **THEN** os testes do worker Python SHALL falhar até o Python ser atualizado
