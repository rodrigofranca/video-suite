# Cortes a partir de uma citação

Até aqui, todo corte vinha da sugestão do modelo de linguagem, e a pessoa só ajustava as bordas (ADR 0024). Na redação, o caminho costuma ser o inverso: alguém sabe a frase que quer, como a declaração do ministro, acha o momento com a busca (ADR 0038) e precisa do corte exato dela. Sem o modelo configurado, ou quando a sugestão falha, não havia como cortar.

**Decisão: o corte nasce da transcrição, palavra por palavra.**

**No painel:**

- A transcrição ganhou **✂ Clip a quote**. Toca-se na primeira palavra da citação e depois na última, em qualquer ordem; um terceiro toque recomeça. Uma transcrição antiga, sem o tempo de cada palavra, escolhe trechos inteiros.
- O corte vai do começo da primeira palavra ao fim da última, com uma folga de 0,15 s antes e 0,25 s depois, porque a fala começa um pouco antes do tempo que o Whisper dá à palavra. A folga nunca alcança a palavra vizinha, então o corte não pega um pedaço da frase ao lado.
- Uma barra mostra a citação, o intervalo e a duração, e um título preenchido com a própria citação, que pode ser mudado. **Preview** toca o intervalo no player, e **Make the clip** cria o corte. Menos de 3 s ou mais de 3 minutos aparece como problema antes de enviar, com as mesmas regras da edição.
- A transcrição também ganhou **Find in the transcript**: acha o texto sem ligar para acentos nem maiúsculas, marca onde está, conta e pula entre os achados.
- O corte novo aparece primeiro na lista, com a etiqueta **Quote** no lugar do número e da nota. A lista de cortes passa a carregar assim que a transcrição termina, mesmo quando a sugestão falhou.

**Na API:** `POST /assets/{id}/clips` recebe `{start, end, title?}`.

- Sem título, o título é o que foi dito no intervalo, cortado numa palavra com "…" depois de 120 caracteres. Num intervalo sem fala, o título é "Clip at m:ss".
- As regras são as da edição: de 3 s a 3 minutos, dentro do vídeo, e 409 enquanto um job do vídeo estiver pendente ou rodando.
- A mesma transação grava o corte e enfileira um `track-faces` só para ele, então o **9:16 face** funciona como nos outros.
- `GET /assets/{id}/transcript` passou a trazer `words`.

**No banco** (migração `0021`):

- `clips.origin` é `suggested` ou `quote`.
- Um corte de citação não tem job nem nota: `job_id` e `score` ficaram opcionais, e uma restrição exige os dois nos cortes sugeridos.
- Na API, `score` vem `null` para uma citação.

**Ordem e novas sugestões:**

- O corte de citação entra com `rank` 1, e os outros descem uma posição: quem escolheu a frase quer vê-la primeiro.
- Sugerir de novo mantém os cortes de citação, como os editados (ADR 0026). As sugestões que repetem mais da metade de um deles ficam de fora.
- Na nova ordem, uma citação vem antes de qualquer nota do modelo. Entre citações, vale a ordem em que foram ditas.
- O aviso antes de sugerir de novo conta as citações entre os cortes "que você fez ou editou".

**O que não muda:** exportar, editar e apagar um corte de citação é igual a qualquer corte.

**Para o futuro:** cortar silêncios e vícios de fala, o próximo item do backlog, usa os mesmos tempos de palavra.
