# Spec Delta

## MODIFIED Requirements

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
