## Context

Fechamento do Milestone 1. A decisão de produto (re-run como nova tentativa) está na ADR 0015.

## Decisions

**Trava por asset:** `CreateJobIfIdle` abre uma transação, trava a linha do asset (`SELECT ... FOR UPDATE`), confere se há jobs `pending`/`running` e só então insere. Dois cliques simultâneos não criam duas tentativas.

**Um Dockerfile Go, dois alvos:** `infra/docker/go.Dockerfile` compila API e worker no mesmo estágio (o `go.work` exige os dois módulos presentes) e gera as imagens pelos alvos `api` e `worker`. Runtime Alpine com usuário sem privilégios; o worker instala `ffmpeg` do Alpine, que inclui `ffprobe`, libx264 e aac.

**Dashboard estático:** com `ssr = false`, o build vira arquivos estáticos servidos por nginx com fallback para `index.html`. `PUBLIC_API_URL` é gravado no build (argumento do Dockerfile).

**Mesmas portas da execução local:** API em 8080 e dashboard em 5173, para o CORS da API e do bucket funcionar sem configuração extra. Por isso a demo e a execução a partir do código não rodam ao mesmo tempo.

**Migração como serviço:** `migrate/migrate` roda antes da API e do worker (`service_completed_successfully`), junto com o `storage-init`.

## Risks / Trade-offs

- **Imagem do worker com 220 MB**, quase tudo FFmpeg → aceitável para um worker de mídia.
- **Build da sandbox de desenvolvimento:** o repositório de pacotes do Alpine precisou ser liberado no ambiente remoto; numa máquina comum o build não precisa de nada especial.
