## Context

Decisões na ADR 0027.

## Decisions

**Banco primeiro, arquivos depois:** o vídeo some para todos assim que a transação termina; se o storage falhar no meio, sobram arquivos órfãos, registrados no log com a pasta. A ordem inversa deixaria um vídeo listado sem arquivos.

**Esperar o vídeo ocioso:** a mesma regra de rodar de novo e exportar, para nenhum worker escrever num vídeo apagado. Uploads parados há mais de uma hora contam como abandonados.

**Cascata do banco:** as tabelas já apagam em cascata a partir de `media_assets`; os registros de arquivos de um corte saem pelo `clipId` dos metadados. Os jobs de exportação ficam no histórico.

**Pasta protegida:** `DeletePrefix` recusa qualquer coisa que não seja `assets/<id>/...` terminando em barra.

**Sem lixeira:** o storage local não guarda versões; uma lixeira pediria regras de retenção.

## Risks / Trade-offs

- **Sem desfazer:** a confirmação é o único freio.
- **Arquivos órfãos:** se o storage falhar depois do banco, a limpeza é manual, pela pasta no log.
