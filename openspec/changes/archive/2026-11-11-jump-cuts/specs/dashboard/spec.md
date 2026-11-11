## ADDED Requirements

### Requirement: Cortes secos no dashboard
Cada corte SHALL ter **Jump cuts**, desligado e com o motivo quando a transcrição não tem o tempo das palavras; escolher um preset SHALL marcar ou desmarcar conforme o preset. O formulário do preset SHALL ter **Editing: Jump cuts**, e a lista de presets SHALL dizer "jump cuts". Um arquivo com cortes secos SHALL mostrar a nova duração, como "27 s, jump cuts".

#### Scenario: Exportar com cortes secos
- **WHEN** o usuário marca **Jump cuts** e exporta **Original** de um corte de 30 s com 3 s de pausas
- **THEN** o arquivo SHALL mostrar "27 s, jump cuts"
