# Tasks

## 1. Re-run de jobs

- [x] 1.1 `store.CreateJobIfIdle` com trava por asset, com teste
- [x] 1.2 `POST /assets/{id}/jobs` (202, 400, 404, 409), com teste
- [x] 1.3 Botão **Run again** e tentativas no dashboard; `poll` com `refresh` sem laços duplicados

## 2. Docker

- [x] 2.1 Dockerfiles da API, do worker e do dashboard; `.dockerignore`
- [x] 2.2 Profile `app` no Compose com `migrate`, `api`, `worker` e `dashboard`
- [x] 2.3 `make demo`, `make demo-logs`, `make sample`; `make down` inclui a demo
- [x] 2.4 Subir a pilha com `--wait` e testar no navegador: upload, re-run de miniatura e de HLS, 409 no segundo pedido

## 3. Documentação

- [x] 3.1 `docs/demo.md`, "Quick start" e "Known limitations" no README
- [x] 3.2 ADR 0015, `CONTEXT.md` e nota de conclusão no roadmap
