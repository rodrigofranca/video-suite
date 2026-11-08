# Spec Delta

## ADDED Requirements

### Requirement: Opções de legenda no contrato
`ExportClipInput` SHALL ter `caption_position` e `caption_color`, omitidos quando vazios, com os valores de `CaptionPositions` (`bottom`, `middle`, `top`) e `CaptionColors` (`yellow`, `green`, `pink`, `blue`, `violet`), e `caption_caps` opcional, em que ausente significa o padrão do estilo.

#### Scenario: Exportação sem opções
- **WHEN** um `ExportClipInput` sem opções de legenda é codificado
- **THEN** o JSON SHALL não ter `caption_position`, `caption_color` nem `caption_caps`
