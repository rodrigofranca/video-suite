# Tasks

- [x] 1 Migração 0021: `origin`, `job_id` e `score` opcionais com a restrição dos sugeridos
- [x] 2 Domínio e store: `Clip.Origin`, `Transcript.Said`, `QuoteTitle`, `CreateClipIfIdle` com o `track-faces`; testes
- [x] 3 Worker de IA: a sugestão mantém os cortes de citação, primeiro na ordem; testes
- [x] 4 API: caso de uso `CreateClip`, `POST /assets/{id}/clips`, `origin` e `score` nulo, `words` na transcrição; testes
- [x] 5 Painel: regras em `quote.ts`, escolha das palavras, barra da citação, achar na transcrição, etiqueta **Quote**, cortes sem sugestão; testes das regras
- [x] 6 ADR 0039, README, CONTEXT, docs/README, demo e backlog
- [x] 7 Na pilha Docker, no celular: busca, momento, achar, citação curta recusada, citação escolhida, prévia, corte, exportação 9:16 face e nova sugestão mantendo a citação primeiro
