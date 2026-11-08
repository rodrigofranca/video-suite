# Spec Delta

## ADDED Requirements

### Requirement: Vinhetas no contrato de exportação
`ExportClipInput` SHALL ter `intro` e `outro` opcionais, cada um com as chaves `horizontal` e `vertical` de uma vinheta (ao menos uma), e `preset` com o nome do preset. O pacote SHALL definir `MaxVignetteSeconds` (30).

#### Scenario: Exportação sem preset
- **WHEN** um `ExportClipInput` sem vinhetas é codificado
- **THEN** o JSON SHALL não ter `intro`, `outro` nem `preset`
