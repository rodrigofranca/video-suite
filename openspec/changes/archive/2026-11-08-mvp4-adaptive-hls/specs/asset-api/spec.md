# Spec Delta

## ADDED Requirements

### Requirement: Qualidades no playback
O playback de um asset `ready`, em `GET /assets/{id}/playback` e em `GET /assets/{id}`, SHALL trazer `renditions`, com `name`, `width` e `height` de cada qualidade, da melhor para a pior, e vazio para vídeos codificados numa qualidade só.

#### Scenario: Vídeo antigo
- **WHEN** o artefato `hls` mais recente não tem qualidades nos metadados
- **THEN** `renditions` SHALL ser uma lista vazia
