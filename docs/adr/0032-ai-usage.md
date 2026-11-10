# Uso de IA registrado por chamada ao modelo

Os tokens de cada chamada ao modelo de linguagem só apareciam como texto no log do job ("model claude-haiku-5-5, 1200 tokens in, 300 out"). Não dava para saber quanto um vídeo custou, nem o gasto do mês. Agora cada chamada vira uma linha no banco, e o painel mostra os totais por vídeo e por período.

**Uma linha por chamada:** a tabela `llm_usage` guarda o vídeo, o job, o tipo do job (resumo, cortes, tradução), o modelo que respondeu, os tokens de entrada e de saída, o custo em dólares e a hora. Toda resposta conta, inclusive a que não serviu e precisou de correção, porque também foi paga. Um lote de tradução é uma chamada. O registro acontece num ponto só: o `ask` do worker de IA, por onde passam todos os jobs que usam o modelo, chama `ctx.record_usage`, e o runner grava a linha com o vídeo e o job. Se a gravação falhar, o job segue e o log avisa; o resultado vale mais que a conta.

**O custo vem do gateway:** o LiteLLM calcula o custo de cada resposta e o devolve no cabeçalho `x-litellm-response-cost`. Para isso ele precisa do preço do modelo, que agora está no `infra/litellm/config.yaml`: US$ 0,10 por milhão de tokens de entrada e US$ 0,50 por milhão de saída, o preço da OpenCode para o Claude Haiku 5.5. Trocar de modelo pede trocar o preço junto. Sem cabeçalho, ou com custo zero numa chamada que usou tokens, o custo fica vazio ("sem preço"), e os totais avisam que o valor real é maior. O código não guarda preços, para não ficar desatualizado em silêncio.

**O passado:** a migração lê de volta os logs dos jobs que rodaram antes, uma linha por job com os tokens que o log registrou, sem custo e marcada como `from_log`. Num resumo que precisou de correção, o log só tinha a última chamada; numa tradução, já tinha a soma.

**Vídeos apagados:** as linhas ficam, sem o vídeo e sem o job, para os totais do período não mudarem quando alguém apaga um vídeo. A página de uso as agrupa como "Deleted videos".

**Na API e no painel:** o detalhe do vídeo traz `aiUsage`, com o total e a divisão por tipo de job, e o painel mostra "AI usage" nos detalhes. `GET /usage?days=` (1, 7, 30, 90 ou 0 para tudo) traz o total do período e os grupos por dia (UTC), modelo, ação e vídeo, os 20 que mais gastaram. A página **AI usage** mostra custo, tokens de entrada e de saída e chamadas, um gráfico de tokens por dia com os valores ao passar o mouse, a mesma série como tabela, e as tabelas por ação, modelo e vídeo.

**Fora da conta:** a transcrição (Whisper) e o rastreamento de rostos rodam na própria máquina e não têm custo por chamada.
