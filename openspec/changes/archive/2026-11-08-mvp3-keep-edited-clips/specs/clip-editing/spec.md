# Spec Delta

## ADDED Requirements

### Requirement: Edições sobrevivem a uma nova sugestão
Um corte editado SHALL continuar depois de a sugestão de cortes rodar de novo, com o mesmo id, intervalo, título e exportações.

#### Scenario: Sugerir de novo depois de editar
- **WHEN** o usuário edita um corte e roda **Clips** de novo
- **THEN** o corte editado SHALL continuar na lista com a etiqueta **Edited**
