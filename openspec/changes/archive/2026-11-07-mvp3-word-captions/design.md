## Context

Decisão na ADR 0021.

## Decisions

**Palavras numa coluna própria:** `transcripts.words` guarda `[{start, end, word}]` em ordem. Ficar fora de `segments` mantém leve a resposta de `GET /assets/{id}/transcript`, que não precisa delas. Cada palavra começa no máximo quando a anterior termina e nunca termina antes de começar.

**Frases curtas:** no vertical, até 4 palavras e 22 caracteres; no original, até 7 palavras e 40 caracteres. A frase também quebra em fim de frase (`.`, `!`, `?`, `…`) e em pausas de mais de 0,8 s. Uma palavra entra no corte quando o meio dela está dentro do intervalo.

**Destaque por evento:** o arquivo ASS tem um evento por palavra, com a frase inteira e a palavra dita em amarelo (`\c&H00E6FF&`). Cada evento vai do início da palavra ao início da seguinte; a última fica até 0,4 s depois do fim, ou até a próxima frase começar. A letra é maior que no estilo de linhas, e no vertical fica a 28% da base.

**Fallback explícito:** sem palavras, o job usa o estilo de linhas, registra o motivo no log e grava no artefato o estilo de fato usado (`captionStyle`), que o painel mostra (`CC words` ou `CC lines`).

## Risks / Trade-offs

- **Transcrições antigas:** precisam rodar de novo para ter palavras; isso também refaz as legendas do player.
- **Precisão dos tempos:** depende do Whisper; palavras muito rápidas podem ficar destacadas por menos de um quadro e são puladas.
