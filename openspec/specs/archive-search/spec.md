# archive-search Specification

## Purpose
Define a busca pelo que foi dito em todos os vídeos com transcrição, e como um resultado leva ao momento do vídeo (ADR 0038).

## Requirements

### Requirement: Índice por trecho
Cada trecho com palavras de uma transcrição SHALL virar uma linha de `transcript_segments`, com início, fim, texto e o `tsvector` da configuração do idioma da transcrição. Uma transcrição nova SHALL substituir as linhas do vídeo, e apagar a transcrição ou o vídeo SHALL apagá-las, quem quer que grave a transcrição. A migração SHALL indexar as transcrições que já existiam.

#### Scenario: Transcrição refeita
- **WHEN** um vídeo é transcrito de novo e uma palavra deixa de aparecer na fala
- **THEN** buscar essa palavra SHALL deixar de achar o vídeo

### Requirement: Idioma, acentos e terminações
Cada transcrição SHALL ser indexada com o stemmer do seu idioma, quando o PostgreSQL tem um, e sem acentos; os outros idiomas SHALL usar as palavras como estão, sem acentos. A busca SHALL ler o texto digitado em cada idioma que o acervo tem e achar qualquer uma das leituras.

#### Scenario: Sem acento e no plural
- **WHEN** um vídeo em português diz "a reforma tributária" e a busca é `reformas tributarias`
- **THEN** o vídeo SHALL aparecer, com "reforma" e "tributária" marcadas

### Requirement: Sintaxe da busca
A busca SHALL exigir todas as palavras de um mesmo trecho, aceitar "uma frase" entre aspas, -palavra para deixar de fora e `or` entre alternativas.

#### Scenario: Palavra deixada de fora
- **WHEN** a busca é `ministro -amanhã`
- **THEN** os trechos que dizem "amanhã" SHALL ficar de fora

### Requirement: Resultados
`GET /search?q=&offset=` SHALL devolver os vídeos que casam, o de melhor trecho primeiro e, entre iguais, o mais novo, 20 por página, com `more` quando há outros. Cada vídeo SHALL trazer o vídeo como na lista, o idioma, quantos trechos casam e os 3 melhores, na ordem em que foram ditos, com início, fim e o texto em partes, marcando as palavras que casaram. `searched` SHALL dizer quantos vídeos têm transcrição. A API SHALL responder 400 sem texto, com mais de 200 caracteres ou com `offset` que não seja um número de 0 a 10000.

#### Scenario: Busca vazia
- **WHEN** o pedido é `GET /search?q=%20`
- **THEN** a API SHALL responder 400 "type what to search for"

### Requirement: Ir ao momento
No painel, **Search** SHALL estar no menu, e a lista de vídeos SHALL ter uma caixa de busca. A busca SHALL ficar no endereço (`/search?q=`). Cada momento SHALL abrir o vídeo com `?t=` (segundos) e `&q=`: o player SHALL começar ali, a transcrição SHALL rolar até o trecho e marcá-lo, e o topo SHALL ter "← Results for …", que volta aos resultados.

#### Scenario: Abrir um momento
- **WHEN** o usuário toca num momento em 2:00 de um resultado
- **THEN** o vídeo SHALL abrir parado em 2:00, com o trecho marcado e visível na transcrição
