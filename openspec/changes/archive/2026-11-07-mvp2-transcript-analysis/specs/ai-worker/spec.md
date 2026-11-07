# Spec Delta

## MODIFIED Requirements

### Requirement: Verificações na inicialização
O worker SHALL exigir `REDIS_URL`, `DATABASE_URL` e as variáveis `S3_*` obrigatórias, conferir Redis, PostgreSQL e o bucket, e carregar o modelo Whisper antes de consumir jobs, encerrando com código não-zero e mensagem clara se algo falhar. Com `LLM_BASE_URL` definida, SHALL conferir se o gateway responde e oferece `LLM_MODEL`, apenas avisando no log quando não, porque a transcrição funciona sem o gateway.

#### Scenario: Redis fora do ar
- **WHEN** o Redis não responde na inicialização
- **THEN** o worker SHALL logar `ai worker stopped: could not connect` e encerrar com código 1

#### Scenario: Gateway fora do ar
- **WHEN** o gateway não responde na inicialização
- **THEN** o worker SHALL logar um aviso e continuar consumindo jobs

#### Scenario: Sem gateway configurado
- **WHEN** `LLM_BASE_URL` está vazia
- **THEN** o worker SHALL avisar que os jobs `analyze-transcript` vão falhar e continuar consumindo jobs
