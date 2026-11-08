# HLS em várias qualidades, com o playlist mestre no mesmo endereço

O HLS saía numa qualidade só, a do original. Num celular com conexão fraca, um vídeo 1080p travava; numa boa, não havia ganho em descer. Agora o vídeo sai em várias qualidades, e o player escolhe a que a conexão aguenta, trocando durante a reprodução.

**A escada:** as qualidades seguem o lado menor do vídeo: 1080p, 720p, 480p e 360p. Medir pelo lado menor faz um vídeo em pé ter as mesmas qualidades de um deitado (o "720p" dele tem 720 de largura). O topo é o tamanho do próprio vídeo, limitado a 1080p: nada é ampliado, e 4K desce para 1080p. Uma qualidade a menos de 10% da de cima fica de fora: um vídeo 768p sai em 768p, 480p e 360p, sem um 720p quase igual. Um vídeo menor que 360p sai só no próprio tamanho.

**A codificação:** um só comando FFmpeg decodifica o original uma vez, divide a imagem e escala cada cópia. A qualidade é constante (CRF 21, como antes), com um teto de bitrate por qualidade: 5000, 2800, 1400 e 800 kbps, e valores intermediários para tamanhos fora da escada. O playlist mestre anuncia o `BANDWIDTH` e a `RESOLUTION` de cada qualidade, que o player usa para escolher; o FFmpeg da imagem (8.0) mede o pico real de cada uma, então vídeos fáceis de comprimir anunciam bem menos que o teto. Todas as qualidades têm quadro-chave no início de cada segmento de 6 s, então o player troca de uma para outra sem salto. O áudio AAC vai junto em cada qualidade; vídeos sem som não ganham trilha. A largura e a altura vêm do vídeo como é exibido, o que já leva em conta a rotação de celulares (ADR 0022).

**O endereço:** o playlist mestre fica em `hls/index.m3u8`, o mesmo arquivo do playlist único de antes, e cada qualidade numa pasta (`hls/720p/index.m3u8` e os segmentos). Assim a API, o player e os vídeos já processados continuam iguais: os antigos seguem tocando na qualidade única até o HLS rodar de novo.

**O envio:** primeiro vão os segmentos, depois o playlist de cada qualidade e, por último, o mestre, para um player nunca ver um playlist que aponta para arquivo que ainda não existe. Em seguida, o worker apaga da pasta `hls/` tudo o que esta codificação não escreveu: os arquivos da qualidade única antiga, ou segmentos a mais de uma codificação anterior. Isso resolve a limitação de "segmentos que sobravam" ao rodar o HLS de novo.

**O custo:** cada qualidade a mais é mais uma codificação, embora as menores sejam baratas e o original seja decodificado uma vez só. Num vídeo 1080p de 40 s, em 4 núcleos, as quatro qualidades levaram 15 s, contra 6 s de uma só: cerca de 2,4 vezes.

**No painel:** o player ganhou o seletor **Quality**, com **Auto** (mostrando a qualidade em uso) e cada qualidade, e os detalhes do vídeo mostram as qualidades disponíveis. A API devolve a lista em `playback.renditions`, que vem vazia para vídeos codificados antes desta mudança. No Safari do iPhone, que toca HLS sozinho sem o hls.js, a troca é automática e o seletor não aparece.
