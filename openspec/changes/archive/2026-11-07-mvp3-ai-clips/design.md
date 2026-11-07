## Context

Primeira entrega da Fase 3. Decisões na ADR 0020.

## Decisions

**Sugerir no pipeline, exportar sob demanda:** sugerir custa uma chamada barata ao modelo e é útil para todo vídeo com fala, então roda sozinho depois da análise. Exportar gasta CPU e armazenamento e depende do que a pessoa quer, então só roda quando pedido, como um job comum, com logs, progresso e histórico.

**Trechos encaixados na transcrição:** o modelo vê cada trecho do Whisper como `[início-fim] texto`, em segundos, e escolhe início e fim. O worker encaixa o início no começo de trecho mais próximo e o fim no final de trecho mais próximo, descarta cortes fora de 10 a 90 s e cortes que repetem mais da metade de um corte melhor, e guarda no máximo 5, por nota. Assim o corte começa e termina em frases inteiras.

**Formatos:** `original` mantém o formato do vídeo (até 1920 px no lado maior); `vertical` recorta o centro em 9:16 e escala para 1080x1920; `vertical-blur` coloca o quadro inteiro sobre uma cópia ampliada e desfocada. Seguir o rosto de quem fala fica para uma próxima entrega.

**Legendas queimadas:** o worker lê o WebVTT do asset, recorta o trecho do corte, divide as legendas em pedaços menores no vertical (até 32 caracteres, tempo proporcional) e escreve um arquivo ASS com fonte DejaVu Sans em negrito, contorno escuro e margem maior no vertical, longe dos botões dos apps. O FFmpeg aplica com o filtro `ass`. Sem legendas no trecho, o corte sai sem elas e o log diz por quê.

**Corte preciso:** `-ss` antes da entrada com recodificação é rápido e preciso no quadro, e os tempos do corte começam em 0, o que as legendas usam.

**Cortes privados:** diferente do HLS, os cortes são arquivos para baixar e publicar, não para o player. Ficam privados e a API redireciona para um link assinado de 15 minutos, com nome de arquivo legível (`<video>-clip-<n>-<formato>.mp4`); com `inline=1`, o link toca no navegador.

**Um job por vez por vídeo:** como as reexecuções, uma exportação só começa quando nenhum outro job do vídeo está pendente ou rodando. O dashboard desativa os botões enquanto isso.

**Código compartilhado no worker de IA:** `answers.py` concentra o pedido de JSON, o pedido de correção e as funções de limpeza usados por análise e cortes.

## Risks / Trade-offs

- **Corte central pode cortar quem fala:** o 9:16 com desfoque é a alternativa segura até haver acompanhamento do rosto.
- **Tempo das legendas no vertical é aproximado:** os pedaços dividem o tempo da legenda pelo tamanho do texto, porque o WebVTT não guarda o tempo de cada palavra.
- **Sugerir de novo apaga os cortes anteriores:** os arquivos já exportados continuam no storage, mas somem da lista.
- **Exportar ocupa o worker Go:** um corte de 18 s em 9:16 com desfoque levou 12 s; vídeos longos na fila esperam.
