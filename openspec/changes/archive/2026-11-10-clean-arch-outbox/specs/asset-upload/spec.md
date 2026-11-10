## MODIFIED Requirements

### Requirement: Criação do asset e do pipeline
Após o arquivo ser gravado, a API SHALL registrar o source e um job `pending` por passo do pipeline, mudar o asset para `processing` e pôr apenas o primeiro job na fila pela outbox (ADR 0034), tudo numa única transação.

#### Scenario: Jobs criados e primeiro enfileirado
- **WHEN** o upload termina com sucesso
- **THEN** o asset SHALL ter os jobs `extract-metadata`, `generate-thumbnail`, `transcode-hls`, `extract-audio`, `transcribe-audio`, `analyze-transcript`, `suggest-clips` e `track-faces` em `pending`, nessa ordem, e só `extract-metadata` SHALL estar na fila

#### Scenario: Fila indisponível
- **WHEN** o Redis está fora do ar durante o upload
- **THEN** a API SHALL responder 201, e o primeiro job SHALL continuar `pending` até o relay conseguir mandá-lo
