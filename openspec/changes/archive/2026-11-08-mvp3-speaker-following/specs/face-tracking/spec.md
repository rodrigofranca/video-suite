# Spec Delta

## MODIFIED Requirements

### Requirement: Rastreio do rosto nos cortes
O job SHALL ler o vídeo por um link assinado, decodificar só os trechos dos cortes do asset, 5 quadros por segundo, achar os rostos com o YuNet numa revisão fixa e, por corte, gravar em `face_track` a fração das amostras com rosto (`coverage`), o caminho da janela (`points`, com `t` desde o início do corte e `x` como fração da largura), o tamanho do quadro, o que segue (`follows`) e quantas pessoas aparecem juntas (`people`). Sem cortes, SHALL terminar sem ler o vídeo.

#### Scenario: Rosto atravessando o quadro
- **WHEN** um corte de 6 s tem um rosto indo da esquerda para a direita
- **THEN** o corte SHALL ter `coverage` acima de 0,8, 30 amostras no log e pontos com `x` crescente, de menos de 0,3 a mais de 0,65

#### Scenario: Vídeo sem cortes
- **WHEN** o asset não tem cortes sugeridos
- **THEN** o job SHALL terminar `done` com o log `nothing to track`

### Requirement: Rosto principal
Os rostos SHALL ser ligados em trilhas, uma por pessoa, pela sobreposição das caixas entre amostras (pelo menos 0,3), tolerando até 3 amostras sem a pessoa e descartando trilhas com menos de 3 amostras. Sem seguir quem fala, o rosto principal SHALL ser o de maior área, com peso 0,6 para rostos a mais de 15% da largura de onde o principal estava. Amostras sem rosto SHALL repetir a última posição, e uma mediana de 5 SHALL remover saltos isolados.

#### Scenario: Dois rostos parecidos
- **WHEN** há um rosto um pouco maior longe e outro perto da posição anterior
- **THEN** o rosto perto SHALL continuar sendo o principal

#### Scenario: Pessoa não detectada por um instante
- **WHEN** um rosto some por 2 amostras e volta no mesmo lugar
- **THEN** ele SHALL continuar na mesma trilha, com a posição interpolada nesse intervalo

## ADDED Requirements

### Requirement: Quem fala
Com duas ou mais trilhas juntas em pelo menos 5 amostras e o áudio extraído do asset, o job SHALL decodificar o corte de novo a 25 quadros por segundo, recortar cada rosto em 112x112 cinza como os autores do Light-ASD, calcular o MFCC do som do corte e dar a cada trilha uma nota por amostra, gravando `follows: "speaker"`. Sem som, sem modelo ou com erro, SHALL seguir o maior rosto com `follows: "face"` e o motivo no log.

#### Scenario: Duas pessoas, a fala alternando
- **WHEN** num vídeo com duas pessoas reais, a da direita com o rosto maior, a da esquerda fala de 0 a 7 s, a da direita de 7 a 14 s e a da esquerda de 14 a 20 s
- **THEN** a janela SHALL começar na esquerda, cortar para a direita perto de 7 s e voltar para a esquerda perto de 14 s

#### Scenario: Duas pessoas sem fala
- **WHEN** um corte tem dois rostos parados e um tom no som
- **THEN** o log SHALL dizer `2 people, following who speaks`, o corte SHALL ter `people: 2` e a janela SHALL ficar no rosto maior

### Requirement: Troca de falante
A janela SHALL ir para a trilha com a maior nota acima de 0 e ficar com ela nas pausas; antes de alguém falar, SHALL ficar no maior rosto. Outra pessoa SHALL liderar 3 amostras seguidas antes da troca, a câmera SHALL ficar pelo menos 8 amostras com alguém antes de trocar de novo, e a troca SHALL ser um corte imediato.

#### Scenario: Interrupção curta
- **WHEN** a outra pessoa tem a maior nota por só 2 amostras
- **THEN** a janela SHALL continuar onde estava

#### Scenario: Volta rápida
- **WHEN** a câmera acabou de cortar para alguém e a pessoa anterior volta a falar logo depois
- **THEN** a câmera SHALL voltar só depois de 8 amostras na pessoa nova

### Requirement: Modelo de falante
Os pesos do Light-ASD ajustados no TalkSet SHALL ser baixados uma vez do repositório dos autores num commit fixo, conferidos por SHA-256 e guardados no cache de modelos. O arquivo SHALL ser lido sem PyTorch, recusando qualquer nome fora do formato de tensores, e a rede SHALL rodar no ONNX Runtime. O MFCC SHALL ser igual ao do `python_speech_features`.

#### Scenario: Arquivo adulterado
- **WHEN** o arquivo baixado não confere com o SHA-256
- **THEN** o download SHALL falhar sem guardar o arquivo

#### Scenario: Pickle com código
- **WHEN** o arquivo de pesos referencia `os.system`
- **THEN** a leitura SHALL falhar com `UnpicklingError`
