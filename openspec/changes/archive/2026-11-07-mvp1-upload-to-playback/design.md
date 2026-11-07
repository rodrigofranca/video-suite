## Context

Primeira fatia de ponta a ponta do MVP1: upload → HLS → player. As decisões de produto estão nas ADRs 0009–0012; aqui ficam as escolhas de implementação.

## Goals / Non-Goals

**Goals:**
- Um MP4 enviado pelo dashboard fica assistível sem nenhum comando manual.
- Status e logs de cada job visíveis no dashboard.
- Nenhum arquivo inteiro em memória: upload em streaming e multipart para o storage.

**Non-Goals:**
- Metadata (ffprobe) e thumbnail reais, progresso percentual durante o FFmpeg, ABR, autenticação.

## Decisions

**Upload em streaming com validação antecipada:** a API lê os 12 primeiros bytes da parte `file` para conferir a caixa `ftyp` antes de criar qualquer registro, depois envia `cabeçalho + resto` ao storage por um leitor que conta bytes, calcula SHA-256 e corta em 2 GB. Se algo falha, o objeto e o asset são apagados (o asset nunca fica em `uploading` órfão).

**`transfermanager` em vez de `feature/s3/manager`:** o pacote antigo foi marcado como obsoleto pela AWS. O novo ainda é v0.x, por isso fica isolado em `storage.go`. Checksums automáticos do SDK ficam em "quando exigido" para funcionar com qualquer servidor S3.

**Logs do job via `slog.Handler`:** os handlers recebem um `*slog.Logger` que grava cada linha em `job_logs` e também na saída do worker, então o mesmo log serve para o terminal e para o dashboard.

**`queue.ErrRetry`:** quando o worker é encerrado no meio de um job (ou o banco está fora), a mensagem não é ackada e o job fica `running`; ele roda de novo no próximo start. A leitura de pendentes pagina pelo ID para não entrar em loop com mensagens deixadas de propósito.

**Ordem dos jobs:** as três linhas recebem `created_at` com 1 ms de diferença para listar sempre na ordem do pipeline.

**HLS:** `libx264 -preset veryfast -crf 21`, `yuv420p`, dimensões pares, keyframe a cada 6 s (`-force_key_frames`), AAC 128k estéreo, áudio opcional. Segmentos sobem antes da playlist, para um player nunca ver uma playlist apontando para segmentos que ainda não existem.

**Dashboard SPA:** `ssr = false`; tudo roda no navegador e fala com a API por `PUBLIC_API_URL`. Upload por XHR para ter progresso.

## Risks / Trade-offs

- **Metadata e thumbnail aparecem como `done`** mesmo sendo stubs → o log do job diz `not implemented yet`; os handlers reais vêm na próxima etapa.
- **Entrega duplicada pode enfileirar o próximo job duas vezes** → a segunda execução encontra o job já concluído e é ignorada.
- **Sem progresso durante o FFmpeg** → vídeos longos ficam em `running` sem porcentagem; `-progress` do FFmpeg fica para o hardening.
- **Job pendente sem mensagem** (Redis caiu entre gravar e enfileirar) → API e worker marcam o job e o asset como `failed` em vez de deixá-los parados.
