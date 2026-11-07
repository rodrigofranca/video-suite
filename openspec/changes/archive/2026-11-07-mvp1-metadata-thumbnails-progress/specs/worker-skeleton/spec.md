# Spec Delta

## REMOVED Requirements

### Requirement: Handlers do MVP1 são stubs que logam e retornam nil
**Reason**: Os três jobs do MVP1 agora têm handlers reais.
**Migration**: Ver as specs `metadata-extraction`, `thumbnail-generation` e `hls-transcoding`.

## MODIFIED Requirements

### Requirement: Worker verifica dependências na inicialização
O worker SHALL exigir `REDIS_URL`, `DATABASE_URL` e as variáveis `S3_*` obrigatórias, encontrar `ffmpeg` e `ffprobe` no `PATH` e conferir Redis, PostgreSQL e o bucket antes de consumir jobs.

#### Scenario: FFmpeg ausente
- **WHEN** o worker inicia numa máquina sem FFmpeg
- **THEN** SHALL logar `ffmpeg not found in PATH` e encerrar com código não-zero
