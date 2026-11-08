## Context

Decisões na ADR 0029.

## Decisions

**Traduzir a transcrição inteira, uma vez por idioma:** a tradução serve para todos os cortes e formatos e dá ao player a legenda do vídeo todo. Num vídeo de uma hora são uns 20 pedidos ao modelo.

**Fala por fala, em lotes:** o modelo recebe falas numeradas (até 40 e cerca de 2.500 caracteres por pedido, com as duas anteriores de contexto) e devolve a mesma quantidade, na mesma ordem. Cada fala traduzida mantém o tempo da original. Uma contagem errada pede uma correção, uma vez.

**Palavras sobre a fala original:** uma palavra a uma fração da fala traduzida, em caracteres, aparece quando a mesma fração da fala original era dita. As pausas ficam no lugar, e os estilos palavra a palavra funcionam sem tempo próprio das palavras traduzidas.

**Tradução antes, exportação depois:** a API cria os dois jobs na mesma transação; a exportação fica pendente e o worker de IA a põe na fila quando a tradução termina (`After`). Uma tradução que falha faz a exportação falhar com o motivo, em vez de sair na língua original.

**Idiomas fixos do alfabeto latino:** a fonte das legendas não cobre japonês, chinês e coreano.

## Risks / Trade-offs

- **O destaque anda com a frase, não com cada palavra**, porque os idiomas ordenam as palavras de outro jeito.
- **Um arquivo por formato:** o mesmo formato em outro idioma substitui o arquivo anterior.
- **Custo do modelo:** a tradução custa mais que um resumo, porque devolve a transcrição inteira; continua centavos por hora de fala com um modelo pequeno.
