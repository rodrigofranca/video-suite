# Spec Delta

## ADDED Requirements

### Requirement: Confirmação ao sugerir de novo
Quando o vídeo tem cortes, **Run again** no job de cortes SHALL primeiro mostrar no próprio job o que vai mudar, quantos cortes serão substituídos e quantos editados ficam, com **Yes, run again** e **Cancel**. Os outros jobs SHALL rodar de novo sem confirmação.

#### Scenario: Um corte editado entre três
- **WHEN** o vídeo tem três cortes, um deles editado, e o usuário toca **Run again** em **Clips**
- **THEN** o job SHALL mostrar `This replaces the other 2 clips with new suggestions. The 1 clip you edited stays.` e só rodar depois de **Yes, run again**
