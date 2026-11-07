# Spec Delta

## Purpose
Define o job `track-faces`, que acha o rosto principal em cada corte sugerido e grava por onde a janela 9:16 deve andar para mantê-lo no quadro.

## ADDED Requirements

### Requirement: Rastreio do rosto nos cortes
O job SHALL ler o vídeo por um link assinado, decodificar só os trechos dos cortes do asset, 5 quadros por segundo, achar os rostos com o YuNet numa revisão fixa e, por corte, gravar em `face_track` a fração das amostras com rosto (`coverage`), o caminho da janela (`points`, com `t` desde o início do corte e `x` como fração da largura) e o tamanho do quadro. Sem cortes, SHALL terminar sem ler o vídeo.

#### Scenario: Rosto atravessando o quadro
- **WHEN** um corte de 6 s tem um rosto indo da esquerda para a direita
- **THEN** o corte SHALL ter `coverage` acima de 0,8, 30 amostras no log e pontos com `x` crescente, de menos de 0,3 a mais de 0,65

#### Scenario: Vídeo sem cortes
- **WHEN** o asset não tem cortes sugeridos
- **THEN** o job SHALL terminar `done` com o log `nothing to track`

### Requirement: Rosto principal
O rosto principal SHALL ser o de maior área, com peso menor (0,6) para rostos a mais de 15% da largura de onde o principal estava. Amostras sem rosto SHALL repetir a última posição vista, e uma mediana de 5 amostras SHALL remover saltos isolados.

#### Scenario: Dois rostos parecidos
- **WHEN** há um rosto um pouco maior longe e outro perto da posição anterior
- **THEN** o rosto perto SHALL continuar sendo o principal

### Requirement: Caminho de câmera
A janela SHALL ficar parada enquanto o rosto está a até 12% da largura da janela do seu centro, andar no máximo 2 larguras de janela por segundo, nunca sair do quadro, e cortar direto para o rosto quando ele fica fora da janela por 3 amostras seguidas. O caminho SHALL ser simplificado com tolerância de 0,4% da largura.

#### Scenario: Rosto parado
- **WHEN** o rosto oscila pouco em torno do centro
- **THEN** o caminho SHALL ter só dois pontos, na mesma posição

#### Scenario: Outra pessoa
- **WHEN** o rosto principal passa de um lado do quadro para o outro e fica lá
- **THEN** a janela SHALL mudar de posição de uma vez, em menos de 0,05 s

### Requirement: Sem caminho quando não ajuda
Com rosto em menos de 30% das amostras, ou com o quadro já mais estreito que 9:16, `points` SHALL ficar vazio, mantendo `coverage` e o tamanho do quadro.

#### Scenario: Vídeo vertical
- **WHEN** o vídeo tem 720x1280
- **THEN** `points` SHALL ficar vazio

### Requirement: Vídeos girados
O quadro SHALL ser girado pela rotação de exibição do vídeo antes da detecção, como o FFmpeg faz, e o tamanho gravado SHALL ser o exibido.

#### Scenario: Vídeo de celular de cabeça para baixo
- **WHEN** o vídeo é gravado girado 180° com a rotação de exibição de 180°
- **THEN** o rosto SHALL ser achado e o caminho SHALL seguir o rosto como exibido
