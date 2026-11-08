## Context

Continuação do 9:16 seguindo o rosto (ADR 0022). Decisões na ADR 0023.

## Decisions

**Modelo audiovisual, não só a boca:** quem ouve também mexe a boca. O Light-ASD (CVPR 2023, MIT, 1,0 M parâmetros) compara o rosto com o som. Os pesos ajustados no TalkSet separaram melhor quem fala de quem ouve que os treinados em filmes (AVA): cerca de +4 contra −2, num vídeo de teste com duas pessoas reais.

**Sem PyTorch:** os pesos oficiais vêm do GitHub dos autores, num commit fixo, conferidos por SHA-256. O arquivo é lido por um unpickler que só aceita os quatro nomes do formato de tensores. A rede vira um grafo ONNX em memória, com as batch norms incorporadas, e roda no ONNX Runtime. Diferença máxima para uma reimplementação em PyTorch nas mesmas entradas: 0,000003. O MFCC segue o `python_speech_features`.

**Só quando precisa:** com uma pessoa por vez, ou sem som, vale o maior rosto, sem custo extra. Com duas pessoas juntas por pelo menos 1 s, o corte é decodificado de novo a 25 quadros por segundo para recortar os rostos, e o som vem do áudio já extraído.

**Janelas com contexto:** as camadas iniciais rodam em janelas de 4 s com 8 quadros de contexto de cada lado, mais que o alcance delas: o mesmo resultado de uma passada única, com memória limitada. A parte final roda em trechos de 1 a 6 s, com a média, como os autores.

**Troca de falante:** a maior nota acima de 0 ganha; nas pausas, a câmera fica com quem falou por último. Outra pessoa precisa liderar 3 amostras (0,6 s) e a câmera fica pelo menos 8 amostras (1,6 s) com alguém. Como a troca já foi confirmada, o corte é imediato.

**Refazer junto:** `jobcontracts.RerunWith` diz que `suggest-clips` refeito leva junto `track-faces`. A API cria os dois na mesma transação; o worker já enfileira o próximo `pending` na ordem do pipeline.

## Risks / Trade-offs

- **Falas ao mesmo tempo:** a câmera pode ficar com a pessoa errada quando as duas falam, ou quando quem ouve mexe muito a boca.
- **Download no primeiro uso:** o worker precisa do GitHub na primeira vez (4 MB); sem ele, segue o maior rosto e avisa no log.
- **Custo:** cerca de 7 s para 20 s de vídeo com duas pessoas, no CPU.
