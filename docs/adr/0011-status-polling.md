# O dashboard acompanha o status por polling

A lista de vídeos consulta a API a cada 3 segundos, e a página de um vídeo consulta enquanto ele não está `ready` ou `failed`. WebSockets e server-sent events foram considerados, mas exigem conexões longas na API e um canal entre worker e API (o worker só escreve no PostgreSQL). Com jobs que levam de segundos a minutos, um atraso de até 3 segundos não atrapalha, e o polling não precisa de nenhuma peça nova. Server-sent events podem entrar depois, se a carga ou a latência justificarem.
