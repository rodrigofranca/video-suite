# Spec Delta

## MODIFIED Requirements

### Requirement: Jobs de cortes
O pacote SHALL definir `suggest-clips` (`SuggestClipsInput`: `asset_id`) e `track-faces` (`TrackFacesInput`: `asset_id`, `storage_key`, `clip_id` opcional) no `Pipeline`, e `export-clip` (`ExportClipInput`: `asset_id`, `clip_id`, `storage_key`, `format`, `captions`, `caption_style`, `output_key`) fora dele, com os formatos `original`, `vertical`, `vertical-face` e `vertical-blur` em `ClipFormats`.

#### Scenario: Exportação fora do pipeline
- **WHEN** `NextJobType("export-clip")` é chamado
- **THEN** SHALL indicar que não há próximo passo

#### Scenario: Rastreio de todos os cortes
- **WHEN** um `TrackFacesInput` é codificado sem `clip_id`
- **THEN** o JSON SHALL não ter o campo `clip_id`
