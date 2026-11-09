# Spec Delta

## ADDED Requirements

### Requirement: Marca d'água no contrato de exportação
`ExportClipInput` SHALL ter `watermark` opcional, com `key`, `position`, `size` e `opacity`. O pacote SHALL definir `WatermarkPositions` e os limites de tamanho (5 a 40) e de opacidade (10 a 100).

#### Scenario: Sem marca d'água
- **WHEN** um `ExportClipInput` sem marca d'água é codificado
- **THEN** o JSON SHALL não ter `watermark`
