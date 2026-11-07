# clip-suggestions Specification

## Purpose
Define o job `suggest-clips`, que pede ao modelo de linguagem os momentos do vídeo que funcionam como cortes curtos e independentes.

## Requirements

### Requirement: Sugestão de cortes
O job SHALL enviar ao gateway cada trecho da transcrição com início e fim em segundos e pedir até 5 cortes de 15 a 60 segundos que façam sentido sozinhos, com título, motivo e nota de 1 a 10, no idioma da transcrição. SHALL gravar os cortes em `clips`, substituindo os anteriores do asset, com o modelo que respondeu.

#### Scenario: Vídeo com vários assuntos
- **WHEN** o modelo devolve três cortes válidos
- **THEN** o asset SHALL ter três cortes, ordenados pela nota, com `rank` 1, 2 e 3

#### Scenario: Sugerir de novo
- **WHEN** o job roda outra vez e o modelo não devolve cortes
- **THEN** o asset SHALL ficar sem cortes

### Requirement: Cortes encaixados e filtrados
Cada corte SHALL começar no início de trecho mais próximo e terminar no final de trecho mais próximo. Cortes com menos de 10 ou mais de 90 segundos, sem título, ou que repetem mais da metade de um corte com nota maior SHALL ser descartados. A nota SHALL ficar entre 1 e 10 (5 quando ausente).

#### Scenario: Corte quase igual a outro melhor
- **WHEN** dois cortes compartilham 75% do mais curto
- **THEN** só o de nota maior SHALL ser gravado

#### Scenario: Corte longo demais
- **WHEN** o modelo propõe um corte de 2 minutos
- **THEN** ele SHALL ser descartado

### Requirement: Nada a cortar
Sem transcrição, sem fala, ou com menos de 30 segundos de fala, o job SHALL terminar `done` sem chamar o modelo, deixando o asset sem cortes. Resposta inválida SHALL seguir a mesma regra da análise: um pedido de correção e, depois, falha.

#### Scenario: Fala curta
- **WHEN** a fala dura 20 segundos
- **THEN** o job SHALL terminar sem chamar o modelo, com o log `nothing to clip`
