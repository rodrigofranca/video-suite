# Ajustar início, fim e título de um corte antes de exportar

O modelo acerta o momento, mas nem sempre as bordas: às vezes o corte começa uma frase antes do gancho ou termina no meio de uma ideia, e o título é genérico. Agora cada corte pode ser ajustado no painel, antes de exportar.

A API ganhou `PATCH /assets/{id}/clips/{clipId}` com qualquer um de `start`, `end` (segundos desde o início do vídeo) e `title`. O corte é alterado no lugar, e não copiado: as exportações, as legendas e o rastreio de rostos já leem início e fim do banco quando rodam, então tudo o que vem depois usa o corte novo sem outra mudança. O corte guarda quando foi editado (`edited_at`, migração `0014`), e o painel mostra a etiqueta **Edited**.

As regras são as mesmas do resto do sistema, conferidas na API e repetidas no painel para avisar antes de salvar. Início e fim ficam dentro do vídeo, segundo a duração dos metadados. O corte tem de 3 s a 3 minutos, mais folgado que os 10 a 90 s que pedimos ao modelo, porque quem edita sabe o que quer. O título não pode ficar vazio, tem até 120 caracteres e os espaços repetidos viram um. Como nas exportações, a edição espera nenhum job do vídeo estar pendente ou rodando (409), para não mudar o corte no meio de uma exportação.

Mudar início ou fim invalida o caminho de câmera do corte, que é calculado sobre o intervalo. A mesma transação limpa `face_track` e cria um `track-faces` só para esse corte (campo novo e opcional `clip_id` na entrada do job); a API o enfileira em seguida. Até ele terminar, o painel mostra "Finding faces…" e as exportações esperam. Mudar só o título não toca no rastreio.

Exportações feitas antes da edição continuam no storage e na lista, mas a API marca o arquivo como `outdated` quando o início ou o fim gravados no artefato diferem dos atuais, e o painel diz "Made before the edit: export again".

No painel, **✎ Edit** abre um editor no próprio card. O início pula para o começo da frase anterior ou da seguinte, e o fim para o final de uma frase, usando os trechos da transcrição; botões de −1 s e +1 s fazem o ajuste fino. **Preview** toca o intervalo do rascunho no player antes de salvar. Os botões são grandes e ficam em uma grade, para funcionar no celular.

Sugerir os cortes de novo continua substituindo a lista, e com ela as edições; o painel não avisa sobre isso ainda.
