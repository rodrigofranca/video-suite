# Spec Delta

## ADDED Requirements

### Requirement: Progresso dos jobs
O worker SHALL gravar o progresso que um handler reporta em `processing_jobs.progress` enquanto o job está `running`, no máximo uma vez por segundo e só quando o valor muda. Um job `done` SHALL ficar com progresso 100.

#### Scenario: Valores próximos
- **WHEN** o handler reporta 10 e logo depois 20
- **THEN** o banco SHALL mostrar 10 até passar um segundo

#### Scenario: Job pendente
- **WHEN** o progresso é gravado para um job que não está `running`
- **THEN** o valor SHALL ser ignorado
