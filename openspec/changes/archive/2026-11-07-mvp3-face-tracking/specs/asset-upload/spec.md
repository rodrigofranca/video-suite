# Spec Delta

## MODIFIED Requirements

### Requirement: Criação do asset e do pipeline
Após o arquivo ser gravado, a API SHALL registrar o source e um job `pending` por passo do pipeline, mudar o asset para `processing` numa única transação e enfileirar apenas o primeiro job.

#### Scenario: Jobs criados e primeiro enfileirado
- **WHEN** o upload termina com sucesso
- **THEN** o asset SHALL ter os jobs `extract-metadata`, `generate-thumbnail`, `transcode-hls`, `extract-audio`, `transcribe-audio`, `analyze-transcript`, `suggest-clips` e `track-faces` em `pending`, nessa ordem, e só `extract-metadata` SHALL estar na fila

#### Scenario: Fila indisponível
- **WHEN** o primeiro job não pode ser enfileirado
- **THEN** a API SHALL responder 500, marcar o job como `failed` com a mensagem do erro e marcar o asset como `failed`
