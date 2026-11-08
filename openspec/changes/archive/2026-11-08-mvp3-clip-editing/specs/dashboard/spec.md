# Spec Delta

## ADDED Requirements

### Requirement: Editar um corte pelo dashboard
Cada corte SHALL ter **✎ Edit**, que abre no card o título e o início e o fim com décimos de segundo, cada um com **Prev sentence** e **Next sentence** (começos de trecho da transcrição para o início, finais para o fim), **−1 s** e **+1 s**. **Preview** SHALL tocar o rascunho; **Save** SHALL ficar desativado com o motivo enquanto o rascunho quebra uma regra, nada mudou ou um job do vídeo roda.

#### Scenario: Pular para as frases
- **WHEN** o usuário abre o editor de um corte de 0:00 a 0:25, toca **Next sentence** no início, **Prev sentence** no fim e salva
- **THEN** o corte SHALL mostrar o novo intervalo e a etiqueta **Edited**, e uma nova exportação SHALL ter a duração do novo intervalo

### Requirement: Depois de editar no dashboard
Enquanto o novo rastreio roda, **9:16 face** SHALL mostrar `Finding faces…`. Uma exportação feita antes da edição SHALL mostrar `Made before the edit: export again`.

#### Scenario: Exportação antiga
- **WHEN** o corte tinha uma exportação em **9:16 face** e o início muda
- **THEN** essa exportação SHALL mostrar `Made before the edit: export again` até ser exportada de novo
