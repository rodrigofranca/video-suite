## ADDED Requirements

### Requirement: Cortes secos no preset
O preset SHALL guardar se as exportações fazem cortes secos (`jumpCuts`, falso por padrão). Exportar com o preset SHALL fazer cortes secos quando ele diz, a menos que o pedido diga `jumpCuts: false`.

#### Scenario: Preset de Reels
- **WHEN** o preset tem `jumpCuts` e o pedido só traz o formato e o preset
- **THEN** o job de exportação SHALL receber `jump_cuts: true`
