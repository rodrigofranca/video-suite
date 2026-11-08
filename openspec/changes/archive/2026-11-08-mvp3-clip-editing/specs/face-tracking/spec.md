# Spec Delta

## ADDED Requirements

### Requirement: Só o corte editado
Com `clip_id`, o job SHALL rastrear só esse corte do asset, deixando os outros como estão. Se o corte não existe mais, SHALL terminar `done` com o log `nothing to track`. Um `clip_id` que não seja UUID SHALL fazer o job falhar.

#### Scenario: Corte editado
- **WHEN** o asset tem dois cortes sem rastreio e o job recebe o `clip_id` do segundo
- **THEN** só o segundo SHALL ganhar `face_track`
