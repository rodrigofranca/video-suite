# Cortes secos: tirar as pausas e os vícios de fala

No Reels, no TikTok e nos Shorts, o corte comum é seco: as pausas entre as frases somem, e a fala segue sem respiro. Até aqui, um corte exportado tinha todas as pausas do vídeo original. Para tirá-las era preciso um editor de vídeo.

**Decisão: o corte seco é uma opção da exportação, planejada a partir do tempo de cada palavra.**

**O que fica** (`apps/worker/internal/jumpcut`):

- Fica cada palavra do corte, com 0,10 s antes e 0,15 s depois. A fala começa um pouco antes do tempo que o Whisper dá à palavra e termina um pouco depois.
- Duas palavras ficam juntas quando sobra menos de 0,30 s entre elas: é o ritmo da frase, e um corte ali soaria picotado.
- Uma pausa mais longa vira 0,25 s de respiro. Um corte de 30 s com duas pausas de 1,4 s e 2,1 s sai com 27 s.
- Os vícios de fala, sons que não são palavras em nenhum dos idiomas, também saem: "uh", "um", "hmm", "ãh", "hã", "äh", "euh". As letras repetidas não contam, então "Ummm…" é "um".
- Ficam de fora da lista palavras que significam algo em algum idioma: "er" (ele, em alemão), "em" e "é" (em português), "like", "tipo", "you know".
- O plano usa as palavras da transcrição original, mesmo com a legenda em outro idioma: a fala é a mesma.

**Como corta** (`ffmpeg.JumpCuts`, um estágio a mais no grafo do ADR 0035):

- Cada trecho mantido é recortado da mesma imagem e do mesmo som (`trim` e `atrim`), e o filtro `concat` junta os trechos.
- O `concat` mantém imagem e som juntos trecho a trecho, então não há desvio acumulado. No teste, 27,00 s de imagem para 27,02 s de som.
- O som de cada trecho entra e sai em 15 ms (`afade`), para as emendas não estalarem.
- O estágio vem depois da legenda: a legenda é queimada no tempo original e cortada junto com a imagem, então continua presa às palavras sem ser refeita.
- O estágio vem antes das vinhetas, que entram inteiras. O caminho de câmera do 9:16 rosto também roda no tempo original, antes do corte.

**Na API:**

- A exportação aceita `jumpCuts`, e o preset ganhou `jumpCuts` (migração `0022`). Como as outras escolhas, o pedido vence o preset.
- Sem o tempo das palavras, a API responde 409 e pede para transcrever de novo; o worker confere de novo ao rodar.
- Um corte sem fala nenhuma sai inteiro, com um aviso no log.
- O arquivo guarda `jumpCuts`, `keptSeconds` e `fillers`, e a lista de exportações mostra a nova duração: "27 s, jump cuts".

**No painel:**

- Cada corte ganhou **Jump cuts**, ao lado da legenda. Fica desligado, com o motivo, quando a transcrição não tem o tempo das palavras.
- O formulário do preset ganhou **Editing: Jump cuts**, e a lista de presets diz "jump cuts".

**Limites conhecidos:**

- O Whisper limpa a maior parte dos vícios de fala ao transcrever, então quase sempre o que sai são as pausas. Pedir ao Whisper que escreva os vícios mudaria também as legendas, e fica para depois.
- A precisão é a do tempo das palavras. No teste, cada pausa ficou com cerca de 0,6 s de silêncio medido, e não 0,25 s, porque o Whisper às vezes começa a palavra um pouco cedo. Ajustar as bordas pelo volume do som é o próximo passo.
- "Like", "tipo" e repetições ("eu, eu acho") ficam, porque decidir quando são vício pede contexto.
- Como as outras escolhas, exportar de novo o mesmo formato substitui o arquivo.
