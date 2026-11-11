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
A página do vídeo SHALL oferecer **Run again** na tentativa mais recente de cada job `done` ou `failed` que a API marca com `canRerun`, visível direto quando o job falhou e ao abrir o job nos demais casos. O botão SHALL ficar oculto enquanto algum job do asset estiver pendente ou rodando. Quando houver mais de uma tentativa de um tipo, cada uma SHALL mostrar seu número, com as antigas esmaecidas. Os nomes dos jobs SHALL vir do `label` da API.

#### Scenario: Refazer a miniatura
- **WHEN** o usuário abre o job de miniatura e clica em **Run again**
- **THEN** uma nova linha `attempt 2` SHALL aparecer e a página SHALL acompanhar o job até ele terminar

#### Scenario: Tradução
- **WHEN** o usuário abre um job de tradução terminado
- **THEN** **Run again** SHALL não aparecer

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

### Requirement: Qualidade do vídeo no dashboard
Com mais de uma qualidade no stream e `hls.js`, o player SHALL mostrar **Quality**, com **Auto** (e a qualidade em uso) e cada qualidade, da melhor para a pior; escolher uma SHALL fixá-la, e **Auto** SHALL voltar à escolha pela conexão. Os detalhes do vídeo SHALL listar as qualidades.

#### Scenario: Fixar 360p
- **WHEN** o usuário escolhe **360p** num vídeo 1080p
- **THEN** o vídeo SHALL passar a tocar com 360 de altura

### Requirement: Idioma das legendas no dashboard
As opções da legenda de um corte SHALL ter **Language**, com o idioma da transcrição (original) e os outros de `CaptionLanguages`. Num idioma ainda sem tradução, o painel SHALL avisar que a primeira exportação traduz a transcrição com o modelo. O arquivo pronto SHALL dizer o idioma da legenda traduzida, e o player SHALL oferecer a legenda traduzida como "(translated)", depois da original.

#### Scenario: Corte em espanhol
- **WHEN** o usuário escolhe **Spanish** e exporta **9:16 crop** num vídeo em inglês
- **THEN** o arquivo SHALL dizer "Spanish" e o player SHALL ter "Spanish (translated)"

### Requirement: Tradução em andamento no dashboard
Enquanto a tradução de uma exportação roda, o formato SHALL mostrar "Translating into …". A lista de jobs SHALL mostrar a tradução como "Translation · idioma", sem o botão de rodar de novo.

#### Scenario: Esperando a tradução
- **WHEN** a exportação em espanhol está pendente e a tradução roda
- **THEN** o formato SHALL mostrar "Translating into Spanish…"

### Requirement: Página de presets
O painel SHALL ter a página **Presets**, no menu, para enviar, tocar e apagar vinhetas, com aviso antes do envio para arquivos que não são MP4, que passam de 100 MB ou de 30 s. Uma vinheta em uso SHALL mostrar os presets que a usam, com o botão de apagar desligado. A página SHALL criar, editar e apagar presets, com as escolhas de legenda e uma vinheta por formato para a abertura e o encerramento.

#### Scenario: Novo preset
- **WHEN** o usuário cria `My channel` com caixa rosa, abertura 16:9 e encerramento 9:16
- **THEN** a lista SHALL mostrar o preset com a legenda e as vinhetas

### Requirement: Preset nos cortes
Com presets, cada corte SHALL ter o seletor **Preset**. Escolher um SHALL preencher as opções da legenda, que ainda podem ser mudadas, e dizer quais vinhetas entram. A exportação SHALL enviar o preset, e o arquivo pronto SHALL mostrar o preset e se tem abertura e encerramento.

#### Scenario: Exportar com preset
- **WHEN** o usuário escolhe `My channel` e exporta **9:16 crop**
- **THEN** o arquivo SHALL mostrar "My channel (opening + closing)"

### Requirement: Marca d'água no dashboard
A página **Presets** SHALL ter a seção **Watermarks**, para enviar e apagar imagens, com miniatura e tamanho, e o formulário do preset SHALL ter **Watermark**: imagem, canto, tamanho e opacidade, com uma prévia num quadro 9:16 e num 16:9. Com um preset de marca d'água, o aviso do corte SHALL dizer qual imagem entra, e o arquivo pronto SHALL mostrar "watermark".

#### Scenario: Prévia
- **WHEN** o usuário escolhe uma imagem, o canto de baixo à esquerda e 25%
- **THEN** a prévia SHALL mostrar a imagem nesse canto nos dois quadros

### Requirement: Uso de IA no dashboard
Os detalhes do vídeo SHALL mostrar **AI usage**: tokens, custo e a divisão por ação, ou "None yet". A página **AI usage**, no menu, SHALL ter os períodos Today, 7, 30 e 90 dias e All time; os totais de custo, tokens de entrada, tokens de saída e chamadas, com aviso quando há chamadas sem preço ou lidas dos logs; um gráfico de tokens por dia, com os valores de cada dia ao passar o mouse ou focar a barra, e a mesma série como tabela; e tabelas por ação, modelo e vídeo, com link para o vídeo.

#### Scenario: Passar o mouse num dia
- **WHEN** o usuário passa o mouse na barra de um dia
- **THEN** o painel SHALL mostrar os tokens de entrada e de saída, as chamadas e o custo daquele dia

### Requirement: Busca no dashboard
O menu SHALL ter **Search**, e a página **Videos** SHALL ter uma caixa "Search what was said…". A página de busca SHALL mostrar dicas antes da primeira busca. Os resultados SHALL ter, por vídeo, a miniatura, o nome, a data, o idioma e os momentos com o tempo e as palavras marcadas, e quantos momentos a mais o vídeo tem. Sem resultado, SHALL dizer em quantos vídeos com transcrição procurou. **More videos** SHALL trazer a página seguinte.

#### Scenario: Nada encontrado
- **WHEN** a busca não acha nada em 32 vídeos com transcrição
- **THEN** o painel SHALL dizer "Nothing found for “…” in 32 videos with a transcript."

### Requirement: Cortes sem sugestão
A lista de cortes SHALL carregar quando a transcrição termina, mesmo que a sugestão de cortes tenha falhado, e as mensagens de falha e de "nenhum corte" SHALL lembrar que dá para fazer um corte de uma citação.

#### Scenario: Sugestão falhou
- **WHEN** a sugestão falha e o usuário faz um corte de citação
- **THEN** a seção **Clips** SHALL mostrar o corte

### Requirement: Cortes secos no dashboard
Cada corte SHALL ter **Jump cuts**, desligado e com o motivo quando a transcrição não tem o tempo das palavras; escolher um preset SHALL marcar ou desmarcar conforme o preset. O formulário do preset SHALL ter **Editing: Jump cuts**, e a lista de presets SHALL dizer "jump cuts". Um arquivo com cortes secos SHALL mostrar a nova duração, como "27 s, jump cuts".

#### Scenario: Exportar com cortes secos
- **WHEN** o usuário marca **Jump cuts** e exporta **Original** de um corte de 30 s com 3 s de pausas
- **THEN** o arquivo SHALL mostrar "27 s, jump cuts"
