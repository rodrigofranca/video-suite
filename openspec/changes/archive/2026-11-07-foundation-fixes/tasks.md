# Tasks

## 1. Repositório e Git

- [x] 1.1 Ancorar as regras de binário no `.gitignore` (`/apps/api/api`, `/apps/worker/worker`) e confirmar com `git check-ignore` que `apps/worker` não é mais ignorado
- [x] 1.2 Converter `openspec/specs/*` para o formato `## Purpose` + `## Requirements` e validar com `openspec validate --specs --strict`

## 2. Infraestrutura local

- [x] 2.1 Substituir `minio/minio` por `rustfs/rustfs:1.0.4` no Compose, com healthchecks em todos os serviços
- [x] 2.2 Criar serviço `storage-init` (`amazon/aws-cli`) que cria o bucket `ultramedia` e aplica `infra/storage/cors.json`
- [x] 2.3 Renomear variáveis `MINIO_*` para `S3_*` no `.env.example`
- [x] 2.4 Corrigir o Makefile: `-include .env`, `migrate` via `go run`, `migrate-down -all`, `env-check`, `build`, `test`
- [x] 2.5 Verificar `make up` duas vezes seguidas e o CORS com `Origin: http://localhost:5173`

## 3. Banco de dados e domínio

- [x] 3.1 Mover `apps/api/internal/domain` e `apps/api/internal/db` para o novo módulo `packages/media-core` e incluí-lo no `go.work`
- [x] 3.2 Trocar `pgx.Conn` por `pgxpool.Pool` com ping de 5 s na inicialização
- [x] 3.3 Adicionar constantes de status de asset e de job
- [x] 3.4 Criar migração `0007_add_lookup_indexes` e verificar `make migrate`, `make migrate-down` e `make migrate` de novo

## 4. Contratos e worker

- [x] 4.1 Definir o formato da mensagem (`NewMessage`, `Values`, `ParseMessage`, `Decode`) em `packages/job-contracts`, com testes
- [x] 4.2 Recriar `apps/worker`: dispatcher por tipo, handlers stub, consumo via XREADGROUP, XACK sempre
- [x] 4.3 Retomar mensagens pendentes do próprio consumidor ao iniciar
- [x] 4.4 Tratar panic de handler como erro e encerrar com SIGINT/SIGTERM
- [x] 4.5 Testes unitários do dispatcher e testes de integração do consumidor com Redis real (incluindo retomada após queda)

## 5. API

- [x] 5.1 Extrair as rotas para `internal/httpapi` com testes dos cenários da spec
- [x] 5.2 Adicionar `API_PORT`, `ReadHeaderTimeout` e encerramento gracioso

## 6. Documentação

- [x] 6.1 Escrever `README.md` na raiz com setup local
- [x] 6.2 Registrar ADRs 0004–0008 (as quatro decisões que estavam dentro do PRD e a troca para RustFS)
- [x] 6.3 Criar `docs/README.md` com a ordem de precedência entre documentos e as divergências conhecidas
