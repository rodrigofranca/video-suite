# O dashboard usa hls.js até o player UltraMedia estar empacotado

O `CONTEXT.md` define o UltraMedia como player principal e o `hls.js` como reserva. Como o UltraMedia ainda não está publicado como pacote npm nem como web component, o dashboard começa com `hls.js` (1.7.3), usando o HLS nativo do navegador quando Media Source Extensions não existem (Safari no iOS). O player fica isolado no componente `HlsPlayer.svelte`, então trocar pelo UltraMedia depois muda um único arquivo. A alternativa — esperar o UltraMedia — bloquearia a primeira demo de ponta a ponta.
