## Context

Revisão da fundação em 2026-11-07, depois de meses sem commits. O objetivo é deixar o repositório rodável do zero antes de começar as features do MVP1. Nada aqui implementa FFmpeg, upload ou telas.

## Goals / Non-Goals

**Goals:**
- `make up && make migrate && make dev-api && make dev-worker` funcionam num clone novo.
- Worker conforme a spec `worker-skeleton`, com testes.
- API e worker compartilham domínio, acesso ao banco e formato da fila.

**Non-Goals:**
- Handlers reais de FFmpeg/ffprobe, upload, storage client e dashboard.
- Retentativas com contador e fila de falhas (`jobs:failed`), previstas para a fase de hardening.

## Decisions

**RustFS no lugar do MinIO (ADR 0008):**
A imagem `minio/minio` não está mais no Docker Hub. RustFS 1.0.4 (Apache-2.0) foi testado com criação de bucket, upload multipart de 28 MB, política de leitura pública por prefixo e CORS via `PutBucketCors`. O código só usa a API S3, então trocar de servidor é só mudar o Compose. O bucket e o CORS são criados por um container `amazon/aws-cli` (`storage-init`), usando chamadas S3 padrão que também funcionam na AWS.

**Formato da mensagem em `job-contracts`:**
Campos planos `job_id`, `type`, `payload` (JSON). O `job_id` entra agora porque o worker vai precisar dele para atualizar status e logs no PostgreSQL; adicioná-lo depois quebraria mensagens já enfileiradas. A linha do job no banco continua sendo a fonte da verdade.

**Retomada de pendentes com nome de consumidor estável:**
Ao iniciar, o worker lê o próprio histórico pendente (`XREADGROUP ... 0`) antes de ler mensagens novas (`>`). Com um único worker no MVP, isso cobre a retomada após queda prometida no `CONTEXT.md` sem precisar de `XAUTOCLAIM`. O nome vem de `WORKER_NAME`, com o hostname como padrão.

**XACK sempre, inclusive em erro e panic:**
Mantido como na spec original: falhas ficam registradas no banco (fase seguinte), não na fila. Panic vira erro, para que uma mensagem ruim não derrube o worker em loop.

**`packages/media-core` em vez de `apps/api/internal`:**
A regra de `internal` do Go impede o worker de importar código da API. O módulo novo segue o padrão do `job-contracts` (resolvido pelo `go.work`, sem `replace`).

**go-redis v9.22.0:**
A v9.23.0 exige Go 1.26 e forçaria a troca de toolchain do workspace inteiro.

## Risks / Trade-offs

- **RustFS é mais novo que o MinIO** → o código depende só da API S3; Garage e SeaweedFS ficam como alternativas.
- **Retomada só para o mesmo nome de consumidor** → se o worker mudar de nome (ex.: container recriado com outro hostname), as mensagens pendentes do nome antigo ficam presas. Definir `WORKER_NAME` fixo ou adicionar `XAUTOCLAIM` no hardening.
- **Testes de integração dependem do Redis** → são pulados (skip) quando o Redis não está acessível.
