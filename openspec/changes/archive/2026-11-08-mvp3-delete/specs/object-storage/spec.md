# Spec Delta

## ADDED Requirements

### Requirement: Apagar uma pasta
O cliente SHALL apagar todos os objetos sob um prefixo e dizer quantos apagou, aceitando só a pasta de um asset ou uma pasta dentro dela (`assets/<id>/...`, terminando em barra, sem `..`).

#### Scenario: Prefixo perigoso
- **WHEN** o prefixo é vazio, `assets/`, `assets/x` ou fora de `assets/`
- **THEN** o cliente SHALL recusar sem apagar nada

#### Scenario: Pasta de um corte
- **WHEN** a pasta de um corte é apagada
- **THEN** só os arquivos desse corte SHALL sumir
