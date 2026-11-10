## MODIFIED Requirements

### Requirement: Jobs, logs e artefatos
A API SHALL expor `GET /assets/{id}/jobs`, `GET /assets/{id}/artifacts`, `GET /jobs/{id}` e `GET /jobs/{id}/logs`, com os logs em ordem cronológica. Cada job SHALL trazer `label`, o nome do tipo para pessoas, e `canRerun`, se ele pode rodar de novo à mão, ambos da tabela de tipos de job.

#### Scenario: Logs de um job
- **WHEN** `GET /jobs/{id}/logs` é chamado para um job que já rodou
- **THEN** SHALL retornar `{"logs": [...]}` com nível, mensagem e horário de cada linha

#### Scenario: Nome e "rodar de novo"
- **WHEN** o detalhe de um vídeo recém-enviado é pedido
- **THEN** o job `transcode-hls` SHALL ter `label` "HLS encoding" e `canRerun` verdadeiro
