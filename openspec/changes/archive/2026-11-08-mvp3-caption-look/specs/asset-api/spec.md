# Spec Delta

## ADDED Requirements

### Requirement: Opções de legenda na API
`POST /assets/{id}/clips/{clipId}/exports` SHALL aceitar `captionPosition`, `captionColor` e `captionCaps`, opcionais, respondendo 400 para valores fora das listas, e SHALL repassá-los ao job. Cada exportação pronta com legenda SHALL trazer `captionPosition`, `captionColor` e `captionCaps` usados.

#### Scenario: Cor desconhecida
- **WHEN** a exportação é pedida com `"captionColor": "red"`
- **THEN** a API SHALL responder 400 listando as cores

#### Scenario: Opções de volta
- **WHEN** uma exportação em `box` com `top`, `green` e maiúsculas termina
- **THEN** a lista de cortes SHALL trazer essa exportação com `captionPosition: "top"`, `captionColor: "green"` e `captionCaps: true`
