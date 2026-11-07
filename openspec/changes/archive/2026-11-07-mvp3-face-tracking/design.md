## Context

Segunda metade do pedido de cortes verticais da Fase 3. Decisões na ADR 0022.

## Decisions

**Passo do pipeline, não da exportação:** achar o rosto exige decodificar o vídeo e rodar um modelo, o que é trabalho do worker de IA (Python, OpenCV). Rodar uma vez, logo depois da sugestão, deixa a exportação só com o FFmpeg, como as outras, e permite mostrar antes de exportar quanto do corte tem rosto. Só os trechos dos cortes são decodificados, não o vídeo inteiro.

**YuNet:** detector de rostos do OpenCV, 230 KB, licença MIT, rápido em CPU. Fica fixado numa revisão do repositório oficial no Hugging Face e é baixado no primeiro uso, como o Whisper, para o mesmo cache.

**Amostragem:** 5 quadros por segundo, reduzidos a 640 px de largura. O vídeo é lido por um link assinado, com busca até o início de cada corte. O rosto principal é o maior, com preferência ao que está perto do anterior, para não pular entre duas pessoas de tamanho parecido.

**Caminho de câmera:** buracos são preenchidos com a última posição e uma mediana de 5 remove saltos isolados. A janela 9:16 fica parada enquanto o rosto está a até 12% da largura da janela do seu centro; fora disso, anda até 2 larguras de janela por segundo. Se o rosto sai da janela e fica fora por 3 amostras, a janela corta para ele, em vez de atravessar o quadro devagar. O caminho é simplificado (Ramer-Douglas-Peucker, tolerância 0,4% da largura) e guardado em JSON no corte: `coverage`, `points` (`t` em segundos desde o início do corte, `x` como fração da largura), `width` e `height`.

**Expressão do FFmpeg:** o worker Go transforma os pontos numa expressão de `t` por partes (retas entre os pontos), presa às bordas do quadro, no `x` do filtro `crop`. Não há filtro extra nem segunda passada.

**Centro como fallback:** sem caminho (job não rodou, rosto em menos de 30% do corte, vídeo já mais estreito que 9:16, ou tamanho do vídeo diferente do usado no rastreio), `vertical-face` sai igual a `vertical`, com o motivo no log e `framing: "center"` no artefato.

## Risks / Trade-offs

- **Rosto maior, não quem fala:** em conversas, a janela segue o rosto maior. Escolher quem está falando exigiria cruzar com o áudio; fica no backlog.
- **Vídeos de celular girados:** o PyAV entrega o quadro sem aplicar a rotação, então o rastreio desses vídeos usa o tamanho errado e a exportação cai no centro.
- **Sugerir de novo exige rastrear de novo:** os cortes novos ficam sem caminho até o `track-faces` rodar; rodar a sugestão pelo dashboard não encadeia o rastreio, que deve ser rodado também.
