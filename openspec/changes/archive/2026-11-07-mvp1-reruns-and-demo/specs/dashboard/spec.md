# Spec Delta

## ADDED Requirements

### Requirement: Rodar um job de novo pelo dashboard
A página do vídeo SHALL oferecer **Run again** na tentativa mais recente de cada job `done` ou `failed`, visível direto quando o job falhou e ao abrir o job nos demais casos. O botão SHALL ficar oculto enquanto algum job do asset estiver pendente ou rodando. Quando houver mais de uma tentativa de um tipo, cada uma SHALL mostrar seu número, com as antigas esmaecidas.

#### Scenario: Refazer a miniatura
- **WHEN** o usuário abre o job de miniatura e clica em **Run again**
- **THEN** uma nova linha `attempt 2` SHALL aparecer e a página SHALL acompanhar o job até ele terminar
