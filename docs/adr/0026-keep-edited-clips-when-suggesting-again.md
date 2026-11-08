# Sugerir os cortes de novo mantém os cortes editados

Até aqui, rodar a sugestão de cortes de novo substituía a lista inteira, e com ela os cortes ajustados à mão (ADR 0024), sem aviso. Um aviso só deixaria a escolha entre perder o trabalho ou não sugerir de novo. Em vez disso, a nova sugestão agora preserva o que foi editado e substitui só o resto.

O job `suggest-clips` apaga apenas os cortes com `edited_at` nulo. Os editados continuam com o mesmo id, então suas exportações, o caminho de câmera e a etiqueta **Edited** seguem valendo. Das sugestões novas, ficam de fora as que repetem mais da metade de um corte editado, pela mesma regra de sobreposição que já evita cortes repetidos entre as sugestões. Em seguida, todos são ordenados pela nota; num empate, o editado vem antes, e depois o que começa antes. Quando não há mais o que cortar, porque o vídeo ficou sem fala ou o modelo não devolveu nada, os editados também ficam. O log diz quantos foram mantidos e quantas sugestões foram deixadas de fora.

No painel, **Run again** do job de cortes passou a pedir confirmação dizendo o que vai acontecer: quantos cortes serão substituídos e quantos editados ficam. A confirmação aparece no próprio job, sem janela do navegador. Os outros jobs continuam rodando de novo com um toque.

Isso substitui o último parágrafo da ADR 0024. O método `ReplaceClips` do Go, usado nos testes, também mantém os editados, mas deixa a sobreposição e a ordem para quem chama, como faz o job em Python.
