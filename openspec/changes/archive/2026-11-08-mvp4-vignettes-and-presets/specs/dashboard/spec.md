# Spec Delta

## ADDED Requirements

### Requirement: Página de presets
O painel SHALL ter a página **Presets**, no menu, para enviar, tocar e apagar vinhetas, com aviso antes do envio para arquivos que não são MP4, que passam de 100 MB ou de 30 s. Uma vinheta em uso SHALL mostrar os presets que a usam, com o botão de apagar desligado. A página SHALL criar, editar e apagar presets, com as escolhas de legenda e uma vinheta por formato para a abertura e o encerramento.

#### Scenario: Novo preset
- **WHEN** o usuário cria `My channel` com caixa rosa, abertura 16:9 e encerramento 9:16
- **THEN** a lista SHALL mostrar o preset com a legenda e as vinhetas

### Requirement: Preset nos cortes
Com presets, cada corte SHALL ter o seletor **Preset**. Escolher um SHALL preencher as opções da legenda, que ainda podem ser mudadas, e dizer quais vinhetas entram. A exportação SHALL enviar o preset, e o arquivo pronto SHALL mostrar o preset e se tem abertura e encerramento.

#### Scenario: Exportar com preset
- **WHEN** o usuário escolhe `My channel` e exporta **9:16 crop**
- **THEN** o arquivo SHALL mostrar "My channel (opening + closing)"
