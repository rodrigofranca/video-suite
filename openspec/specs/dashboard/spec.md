# dashboard Specification

## Purpose
Define o dashboard web (SvelteKit) onde o usuário envia vídeos, acompanha os jobs e assiste ao resultado em HLS.

## Requirements

### Requirement: Lista de vídeos
A rota `/assets` SHALL listar os assets com miniatura, duração, nome, resolução, status e data, e SHALL atualizar a cada 3 segundos. A raiz `/` SHALL redirecionar para `/assets`.

#### Scenario: Nenhum vídeo
- **WHEN** não há assets
- **THEN** a página SHALL mostrar um convite para enviar um MP4

#### Scenario: API fora do ar
- **WHEN** a API não responde
- **THEN** a página SHALL dizer que não conseguiu falar com a API e em qual endereço

#### Scenario: Vídeo processado
- **WHEN** um asset tem miniatura e metadados
- **THEN** o item SHALL mostrar a miniatura com a duração por cima e a resolução

### Requirement: Upload com progresso
A rota `/upload` SHALL aceitar um arquivo `.mp4` de até 2 GB, mostrar o progresso do envio e abrir a página do vídeo ao terminar. Erros da API SHALL aparecer na tela.

#### Scenario: Arquivo recusado
- **WHEN** a API recusa o arquivo
- **THEN** a mensagem de erro da API SHALL aparecer abaixo do formulário

### Requirement: Página do vídeo
A rota `/assets/{id}` SHALL mostrar status, os jobs com status e duração (ou porcentagem, enquanto rodam), os logs de cada job ao clicar nele, os metadados técnicos e os detalhes do source. Enquanto o asset não estiver `ready` ou `failed`, a página SHALL atualizar a cada 3 segundos.

#### Scenario: Vídeo em processamento
- **WHEN** o asset está `processing`
- **THEN** o lugar do player SHALL mostrar que o vídeo está sendo processado

#### Scenario: Vídeo com falha
- **WHEN** o asset está `failed`
- **THEN** a página SHALL mostrar o erro do job HLS

#### Scenario: Conversão em andamento
- **WHEN** o job HLS está `running` com progresso
- **THEN** o lugar do player SHALL mostrar a miniatura com `Encoding HLS… N%` e o job SHALL ter uma barra de progresso

#### Scenario: Metadados
- **WHEN** os metadados foram extraídos
- **THEN** os detalhes SHALL mostrar duração, resolução, codecs, frame rate e bitrate, com um link para o JSON completo do ffprobe

### Requirement: Player HLS
Quando o asset estiver `ready`, a página SHALL reproduzir a playlist com `hls.js`, ou com o HLS nativo do navegador quando Media Source Extensions não existirem.

#### Scenario: Reprodução
- **WHEN** o usuário abre um vídeo `ready` e aperta play
- **THEN** o vídeo SHALL tocar a partir da playlist pública

### Requirement: Rodar um job de novo pelo dashboard
A página do vídeo SHALL oferecer **Run again** na tentativa mais recente de cada job `done` ou `failed`, visível direto quando o job falhou e ao abrir o job nos demais casos. O botão SHALL ficar oculto enquanto algum job do asset estiver pendente ou rodando. Quando houver mais de uma tentativa de um tipo, cada uma SHALL mostrar seu número, com as antigas esmaecidas.

#### Scenario: Refazer a miniatura
- **WHEN** o usuário abre o job de miniatura e clica em **Run again**
- **THEN** uma nova linha `attempt 2` SHALL aparecer e a página SHALL acompanhar o job até ele terminar

### Requirement: Transcrição e legendas no dashboard
A página do vídeo SHALL carregar as legendas do asset no player e mostrar uma seção **Transcript** com o idioma, o modelo e os trechos com horário. Clicar num trecho SHALL levar o player àquele ponto, e o trecho em reprodução SHALL ficar destacado. Enquanto a transcrição roda, a seção SHALL mostrar o progresso; se o vídeo não tem áudio ou a transcrição falhou, SHALL dizer isso.

#### Scenario: Clique num trecho
- **WHEN** o usuário clica no trecho que começa em 0:07
- **THEN** o player SHALL ir para cerca de 7 segundos e o trecho SHALL ficar destacado

#### Scenario: Legendas no player
- **WHEN** o vídeo tem legendas em inglês
- **THEN** o player SHALL ter uma faixa de legendas `English` carregada do storage

### Requirement: Resumo, tags e capítulos no dashboard
A página do vídeo SHALL mostrar uma seção **Summary** com o resumo, as tags, os capítulos com horário e o modelo que os escreveu. Clicar num capítulo SHALL levar o player ao início dele, e o capítulo em reprodução SHALL ficar destacado. Enquanto a análise roda, a seção SHALL dizer que o resumo está sendo escrito; se a análise falhou ou o vídeo não tem fala, SHALL dizer isso.

#### Scenario: Clique num capítulo
- **WHEN** o usuário clica no capítulo que começa em 0:49
- **THEN** o player SHALL ir para cerca de 49 segundos e esse capítulo SHALL ficar destacado

#### Scenario: Análise com falha
- **WHEN** o job de resumo falhou
- **THEN** a seção SHALL dizer que o resumo não pôde ser escrito e apontar para o job, que oferece **Run again**

### Requirement: Cortes no dashboard
A página do vídeo SHALL mostrar uma seção **Clips** com cada corte sugerido: posição, título, nota, intervalo, duração e motivo, e um botão **Preview** que toca só o intervalo no player. Enquanto a sugestão roda, falhou ou não achou cortes, a seção SHALL dizer isso.

#### Scenario: Prévia
- **WHEN** o usuário clica em **Preview** num corte que começa em 1:08
- **THEN** o player SHALL tocar a partir de 1:08 e pausar no fim do corte

### Requirement: Exportar cortes pelo dashboard
Cada corte SHALL ter um botão por formato (**Original**, **9:16 crop**, **9:16 face**, **9:16 blur**). Durante a exportação, o formato SHALL mostrar o progresso; ao terminar, o vídeo exportado, o link **Download** e o estilo das legendas. Os botões SHALL ficar desativados enquanto algum job do vídeo estiver pendente ou rodando, e os jobs de exportação SHALL não aparecer na lista de jobs do pipeline.

#### Scenario: Exportar em 9:16 com desfoque
- **WHEN** o usuário clica em **9:16 blur** no primeiro corte
- **THEN** a página SHALL mostrar o progresso e, ao terminar, o vídeo vertical e o link de download

### Requirement: Escolha das legendas no dashboard
Cada corte SHALL oferecer **Captions**: **Highlighted word** (padrão, quando a transcrição tem palavras), **Word box**, **Karaoke**, **One word at a time**, **Lines** (quando o vídeo tem legendas) ou **Off**. Os estilos de palavras SHALL ficar desativados sem palavras na transcrição; nesse caso, se o vídeo tem legendas, a seção SHALL dizer uma vez que basta rodar a transcrição de novo.

#### Scenario: Vídeo transcrito antes das palavras
- **WHEN** o usuário abre um vídeo transcrito sem palavras e roda **Transcription** de novo
- **THEN** antes os estilos de palavras SHALL estar desativados, com o aviso, e depois **Highlighted word** SHALL ser a escolha padrão, sem o aviso

#### Scenario: Exportar em karaokê
- **WHEN** o usuário escolhe **Karaoke** e clica em **9:16 blur**
- **THEN** a exportação SHALL terminar mostrando o estilo **Karaoke**

### Requirement: Rosto no dashboard
Antes de exportar, **9:16 face** SHALL dizer em quanto do corte há rosto (`Face in 85% of the clip`), que segue quem fala (`Follows who speaks · 2 people`), ou que vai ficar no centro porque o rastreio não rodou ou achou poucos rostos. Uma exportação que ficou no centro SHALL mostrar `centred`. A página SHALL buscar os cortes de novo quando o rastreio termina.

#### Scenario: Rastreio terminado
- **WHEN** o job `track-faces` termina com rosto em 100% do primeiro corte
- **THEN** o primeiro corte SHALL mostrar `Face in 100% of the clip` em **9:16 face**, sem recarregar a página

#### Scenario: Duas pessoas
- **WHEN** o primeiro corte segue quem fala, com duas pessoas
- **THEN** **9:16 face** SHALL mostrar `Follows who speaks · 2 people`

### Requirement: Editar um corte pelo dashboard
Cada corte SHALL ter **✎ Edit**, que abre no card o título e o início e o fim com décimos de segundo, cada um com **Prev sentence** e **Next sentence** (começos de trecho da transcrição para o início, finais para o fim), **−1 s** e **+1 s**. **Preview** SHALL tocar o rascunho; **Save** SHALL ficar desativado com o motivo enquanto o rascunho quebra uma regra, nada mudou ou um job do vídeo roda.

#### Scenario: Pular para as frases
- **WHEN** o usuário abre o editor de um corte de 0:00 a 0:25, toca **Next sentence** no início, **Prev sentence** no fim e salva
- **THEN** o corte SHALL mostrar o novo intervalo e a etiqueta **Edited**, e uma nova exportação SHALL ter a duração do novo intervalo

### Requirement: Depois de editar no dashboard
Enquanto o novo rastreio roda, **9:16 face** SHALL mostrar `Finding faces…`. Uma exportação feita antes da edição SHALL mostrar `Made before the edit: export again`.

#### Scenario: Exportação antiga
- **WHEN** o corte tinha uma exportação em **9:16 face** e o início muda
- **THEN** essa exportação SHALL mostrar `Made before the edit: export again` até ser exportada de novo

### Requirement: Aparência das legendas no dashboard
Com legenda escolhida, cada corte SHALL mostrar **Position** (Bottom, Middle, Top), bolinhas de cor só para os estilos com destaque, já na cor do estilo, e **CAPS**, marcado de início só em **One word at a time**. A exportação SHALL usar essas escolhas, e a exportação pronta SHALL resumir o que difere do padrão, como `Word box, top, green, caps`.

#### Scenario: Verde em cima
- **WHEN** o usuário escolhe **Top**, a bolinha verde e exporta em **9:16 crop**
- **THEN** a exportação pronta SHALL mostrar `top` e `green` no resumo e o vídeo SHALL ter a legenda verde em cima

### Requirement: Confirmação ao sugerir de novo
Quando o vídeo tem cortes, **Run again** no job de cortes SHALL primeiro mostrar no próprio job o que vai mudar, quantos cortes serão substituídos e quantos editados ficam, com **Yes, run again** e **Cancel**. Os outros jobs SHALL rodar de novo sem confirmação.

#### Scenario: Um corte editado entre três
- **WHEN** o vídeo tem três cortes, um deles editado, e o usuário toca **Run again** em **Clips**
- **THEN** o job SHALL mostrar `This replaces the other 2 clips with new suggestions. The 1 clip you edited stays.` e só rodar depois de **Yes, run again**

### Requirement: Apagar pelo dashboard
A página do vídeo SHALL ter, no fim, **Delete video**, e cada corte SHALL ter **🗑 Delete**. Os dois SHALL pedir confirmação no próprio lugar, dizendo o que será apagado e que não dá para desfazer, e SHALL ficar desativados enquanto algum job do vídeo roda. Depois de apagar o vídeo, o painel SHALL voltar para a lista.

#### Scenario: Apagar o vídeo
- **WHEN** o usuário toca **Delete video** e confirma com **Yes, delete the video**
- **THEN** o painel SHALL mostrar a lista de vídeos, sem esse vídeo
