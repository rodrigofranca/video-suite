# Spec Delta

## MODIFIED Requirements

### Requirement: Rosto no dashboard
Antes de exportar, **9:16 face** SHALL dizer em quanto do corte há rosto (`Face in 85% of the clip`), que segue quem fala (`Follows who speaks · 2 people`), ou que vai ficar no centro porque o rastreio não rodou ou achou poucos rostos. Uma exportação que ficou no centro SHALL mostrar `centred`. A página SHALL buscar os cortes de novo quando o rastreio termina.

#### Scenario: Rastreio terminado
- **WHEN** o job `track-faces` termina com rosto em 100% do primeiro corte
- **THEN** o primeiro corte SHALL mostrar `Face in 100% of the clip` em **9:16 face**, sem recarregar a página

#### Scenario: Duas pessoas
- **WHEN** o primeiro corte segue quem fala, com duas pessoas
- **THEN** **9:16 face** SHALL mostrar `Follows who speaks · 2 people`
