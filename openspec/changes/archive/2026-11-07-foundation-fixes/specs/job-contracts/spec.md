# Spec Delta

## ADDED Requirements

### Requirement: Formato da mensagem na fila
O pacote SHALL definir o formato das mensagens do stream `jobs` (campos `job_id`, `type` e `payload` com o JSON da struct de entrada), as constantes do stream e do consumer group, e as funções para montar (`NewMessage`, `Values`) e ler (`ParseMessage`, `Decode`) uma mensagem.

#### Scenario: Mensagem faz ida e volta sem perda
- **WHEN** a API monta uma mensagem com `NewMessage` e o worker a lê com `ParseMessage` e `Decode`
- **THEN** o worker SHALL obter o mesmo `job_id`, o mesmo `type` e a mesma struct de entrada

#### Scenario: Mensagem incompleta é rejeitada
- **WHEN** `ParseMessage` recebe campos sem `job_id`, sem `type`, sem `payload` ou com `payload` que não é JSON válido
- **THEN** SHALL retornar erro
