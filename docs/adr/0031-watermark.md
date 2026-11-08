# Marca d'água no preset

Depois das vinhetas (ADR 0030), faltava a marca d'água: o logo do canal num canto do corte, do começo ao fim. Agora ela é mais uma escolha do preset.

**A imagem:** é um PNG, que mantém a transparência, ou um JPEG, de até 5 MB e 8 a 4096 px de lado. Fica numa biblioteca própria, privada como a das vinhetas (`library/watermarks/{id}.png` ou `.jpg`). Diferente das vinhetas, a própria API lê o formato e o tamanho da imagem com a biblioteca padrão do Go, sem FFmpeg, e recusa logo no envio o que não serve. Uma imagem usada por um preset não pode ser apagada.

**No preset:** a imagem, o canto (`top-right`, o padrão, `top-left`, `bottom-right` ou `bottom-left`), o tamanho e a opacidade. O tamanho é a largura do logo como porcentagem do lado menor do corte: 15% são 162 px tanto num 9:16 de 1080x1920 quanto num 16:9 de 1920x1080. Assim, a marca tem o mesmo tamanho nos dois formatos e um preset serve para os dois. A margem até a borda é de 4% do lado menor. O painel oferece 10 a 30% de tamanho e 40 a 100% de opacidade; a API aceita 5 a 40% e 10 a 100%.

**Na exportação:** a API copia para o job a chave da imagem e as escolhas, como faz com as vinhetas, e mudar o preset depois não altera a exportação já pedida. O worker confere a imagem e a põe sobre o corte no mesmo comando FFmpeg (`overlay`, com o alfa multiplicado pela opacidade). Ela entra depois do enquadramento e antes da legenda: a legenda fica por cima e continua legível. As vinhetas ficam sem a marca, porque já são a identidade do canal. O arquivo registra que tem marca d'água.

**No painel:** a página **Presets** ganhou a seção **Watermarks**, para enviar e apagar imagens, com miniatura sobre um fundo quadriculado que mostra a transparência. O formulário do preset ganhou a parte **Watermark**, com uma prévia de como o logo fica num quadro 9:16 e num 16:9. Em cada corte, o aviso do preset diz qual marca entra, e o arquivo pronto mostra "watermark".

**Limitações:** a marca fica parada num canto, sem animação. Nos cortes em pé, os apps sociais desenham botões perto da borda direita e de baixo; um canto de cima costuma ficar livre.
