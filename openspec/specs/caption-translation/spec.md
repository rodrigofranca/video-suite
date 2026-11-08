# caption-translation Specification

## Purpose
Define o job `translate-transcript`, que traduz a transcrição de um vídeo com o modelo de linguagem, para legendas em outro idioma.

## Requirements

### Requirement: Tradução fala por fala
O job SHALL ler a transcrição do vídeo e enviar ao modelo, pelo gateway, as falas não vazias numeradas, em lotes de até 40 falas e cerca de 2.500 caracteres, cada lote com as duas falas anteriores como contexto, pedindo um JSON `{"lines": [...]}` com exatamente uma tradução por fala, na mesma ordem. Cada fala traduzida SHALL manter o início e o fim da original. O português SHALL ser pedido como no Brasil.

#### Scenario: Contagem errada
- **WHEN** o modelo devolve menos falas do que recebeu
- **THEN** o job SHALL pedir uma correção uma vez, dizendo quantas faltam, e falhar se a segunda resposta também não servir

#### Scenario: Transcrição longa
- **WHEN** a transcrição tem 5 falas e o lote é de 2
- **THEN** SHALL haver 3 pedidos, e o segundo SHALL trazer as falas 1 e 2 como contexto

### Requirement: Palavras sobre a fala original
O job SHALL dar a cada palavra traduzida um início e um fim: a palavra que começa a uma fração da fala traduzida, em caracteres, SHALL começar quando a mesma fração das palavras originais daquela fala era dita, e terminar do mesmo modo. Sem tempo das palavras originais, o tempo da fala SHALL ser dividido pelo tamanho das palavras.

#### Scenario: Pausa no meio da fala
- **WHEN** a fala original tem uma pausa antes da última palavra
- **THEN** nenhuma palavra traduzida SHALL começar dentro da pausa

### Requirement: Tradução guardada
O job SHALL guardar em `transcript_translations` uma tradução por vídeo e idioma, substituindo a anterior, com as falas, as palavras, o idioma de origem e o modelo que respondeu, e gravar a legenda WebVTT traduzida em `assets/{id}/subtitles/{idioma}.vtt` com um artefato `subtitles` cujos metadados têm `language` e `translatedFrom`. Uma transcrição nova SHALL apagar as traduções do vídeo na mesma transação.

#### Scenario: Transcrição refeita
- **WHEN** a transcrição de um vídeo traduzido para o inglês é salva de novo
- **THEN** o vídeo SHALL ficar sem traduções

### Requirement: Entrada recusada
O job SHALL falhar, sem chamar o modelo, quando o idioma não está em `CaptionLanguages`, quando é o da transcrição, quando não há transcrição ou fala, ou quando nenhum modelo está configurado.

#### Scenario: Mesmo idioma
- **WHEN** a transcrição está em português e o pedido é de português
- **THEN** o job SHALL falhar dizendo que a transcrição já está nesse idioma
