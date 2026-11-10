## MODIFIED Requirements

### Requirement: Rodar um job de novo
`POST /assets/{id}/jobs` com `{"type": "<tipo>"}` SHALL criar uma nova tentativa `pending` daquele tipo, com a entrada montada a partir do source do asset, enfileirá-la e responder 202 com o job criado. Só os tipos que a tabela de tipos de job marca como "pode rodar de novo" SHALL ser aceitos.

#### Scenario: Nova tentativa de miniatura
- **WHEN** todos os jobs do asset terminaram e o pedido é `{"type": "generate-thumbnail"}`
- **THEN** SHALL existir uma nova linha de job desse tipo, enfileirada, e as tentativas anteriores SHALL continuar no histórico

#### Scenario: Tipo desconhecido ou corpo inválido
- **WHEN** o tipo não é um job do pipeline ou o corpo não é JSON
- **THEN** a API SHALL responder 400

#### Scenario: Job que roda com uma exportação
- **WHEN** o pedido é `{"type": "export-clip"}` ou `{"type": "translate-transcript"}`
- **THEN** a API SHALL responder 400 dizendo que esse job roda quando um corte é exportado
