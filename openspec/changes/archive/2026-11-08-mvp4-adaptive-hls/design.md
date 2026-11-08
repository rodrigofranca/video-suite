## Context

Decisões na ADR 0028.

## Decisions

**Lado menor:** a escada usa o lado menor, então um vídeo em pé tem as mesmas qualidades de um deitado. O topo é o tamanho do original, até 1080p; qualidades a menos de 10% da de cima ficam de fora.

**Um comando, CRF com teto:** decodifica o original uma vez, divide e escala; CRF 21 como antes, com `maxrate` por qualidade (5000, 2800, 1400, 800 kbps) e `bufsize` do dobro. O mestre anuncia `BANDWIDTH` e `RESOLUTION` de cada qualidade, medidos pelo FFmpeg.

**Quadros-chave alinhados:** `force_key_frames` a cada 6 s vale para todas as qualidades, então os segmentos coincidem e a troca não salta.

**Mesmo endereço:** o mestre fica em `hls/index.m3u8`; API, player e vídeos antigos não mudam.

**Ordem de envio e limpeza:** segmentos, playlists das qualidades, mestre; depois, remove o que sobrou de codificações anteriores, sem janela em que o vídeo fica sem arquivos.

## Risks / Trade-offs

- **Mais tempo de codificação:** cerca de 2,4 vezes num vídeo 1080p (15 s contra 6 s para 40 s de vídeo, em 4 núcleos).
- **Só H.264 em MPEG-TS:** sem HEVC/AV1 nem fMP4, por compatibilidade.
