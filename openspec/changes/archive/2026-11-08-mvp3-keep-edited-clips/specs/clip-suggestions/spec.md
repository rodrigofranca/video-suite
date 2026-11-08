# Spec Delta

## MODIFIED Requirements

### Requirement: Sugestão de cortes
O job SHALL enviar ao gateway cada trecho da transcrição com início e fim em segundos e pedir até 5 cortes de 15 a 60 segundos que façam sentido sozinhos, com título, motivo e nota de 1 a 10, no idioma da transcrição. SHALL gravar os cortes em `clips`, com o modelo que respondeu, substituindo os anteriores do asset que não foram editados à mão.

#### Scenario: Vídeo com vários assuntos
- **WHEN** o modelo devolve três cortes válidos
- **THEN** o asset SHALL ter três cortes, ordenados pela nota, com `rank` 1, 2 e 3

#### Scenario: Sugerir de novo
- **WHEN** o job roda outra vez, o asset não tem cortes editados e o modelo não devolve cortes
- **THEN** o asset SHALL ficar sem cortes

## ADDED Requirements

### Requirement: Cortes editados mantidos
Os cortes com `edited_at` SHALL continuar, com o mesmo id, quando o job roda de novo, inclusive sem nada a cortar. Sugestões novas que repetem mais da metade de um corte editado SHALL ficar de fora, e todos os cortes SHALL ser ordenados pela nota, com o editado antes num empate. O log SHALL dizer quantos editados ficaram e quantas sugestões saíram.

#### Scenario: Sugestão repete um editado
- **WHEN** um corte de 60 a 89,5 s foi editado e a nova resposta traz 60 a 79,5 s com nota 9 e 0 a 29,5 s com nota 8
- **THEN** o asset SHALL ter o novo de 0 a 29,5 s com `rank` 1 e o editado com `rank` 2, e o log SHALL dizer `kept 1 clip edited by hand; left out 1 new suggestion that repeated them`

#### Scenario: Nada mais a cortar
- **WHEN** o asset tem um corte editado e o modelo não devolve cortes
- **THEN** o asset SHALL ficar só com o corte editado, com `rank` 1
