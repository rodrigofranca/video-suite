## MODIFIED Requirements

### Requirement: Cortes editados mantidos
Os cortes com `edited_at` e os cortes de citação (`origin` `quote`) SHALL continuar, com o mesmo id, quando o job roda de novo, inclusive sem nada a cortar. Sugestões novas que repetem mais da metade de um deles SHALL ficar de fora, e todos os cortes SHALL ser ordenados pela nota, com o editado antes num empate e os de citação, que não têm nota, antes de todos. O log SHALL dizer quantos cortes feitos ou editados à mão ficaram e quantas sugestões saíram.

#### Scenario: Sugestão repete um editado
- **WHEN** um corte de 60 a 89,5 s foi editado e a nova resposta traz 60 a 79,5 s com nota 9 e 0 a 29,5 s com nota 8
- **THEN** o asset SHALL ter o novo de 0 a 29,5 s com `rank` 1 e o editado com `rank` 2, e o log SHALL dizer `kept 1 clip made or edited by hand; left out 1 new suggestion that repeated them`

#### Scenario: Nada mais a cortar
- **WHEN** o asset tem um corte editado e o modelo não devolve cortes
- **THEN** o asset SHALL ficar só com o corte editado, com `rank` 1
