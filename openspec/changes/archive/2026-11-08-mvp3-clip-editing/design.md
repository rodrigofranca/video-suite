## Context

Decisões na ADR 0024.

## Decisions

**Editar no lugar:** exportações, legendas e rastreio já leem início e fim do banco quando rodam, então mudar o corte basta. `edited_at` registra a edição.

**Regras na API e no painel:** a API é quem decide (400 com o motivo); o painel repete as mesmas regras para avisar antes de salvar. Limites de 3 s a 3 minutos, mais folgados que os 10 a 90 s pedidos ao modelo.

**Esperar o vídeo ocioso:** como exportar e rodar de novo, editar devolve 409 enquanto algum job do vídeo está pendente ou rodando, para não mudar um corte no meio de uma exportação.

**Rastrear só o corte editado:** a mesma transação muda o corte, limpa `face_track` e cria o `track-faces` com `clip_id`; a API o enfileira depois. Mudar só o título não refaz nada.

**Exportações antigas ficam:** o arquivo continua no storage; a API compara início e fim gravados no artefato com os atuais e marca `outdated`.

**Frases da transcrição:** o início pula para o começo de um trecho e o fim para o final de um trecho, como os cortes sugeridos.

## Risks / Trade-offs

- **Sugerir de novo apaga as edições:** a lista é substituída; avisar antes ficou no backlog.
- **Edições simultâneas:** a última ganha; não há controle de versão do corte.
