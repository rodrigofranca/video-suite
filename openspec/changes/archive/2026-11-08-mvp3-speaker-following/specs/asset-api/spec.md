# Spec Delta

## ADDED Requirements

### Requirement: Rosto e falante nos cortes
Cada corte de `GET /assets/{id}/clips` SHALL ter também `faceFollows` (`speaker` ou `face`) e `facePeople` (pessoas juntas na tela), nulos antes do rastreio. Caminhos gravados antes de seguir quem fala SHALL aparecer como `face`, com `facePeople` nulo.

#### Scenario: Corte com duas pessoas
- **WHEN** o caminho do corte tem `follows: "speaker"` e `people: 2`
- **THEN** o corte SHALL trazer `faceFollows: "speaker"` e `facePeople: 2`
