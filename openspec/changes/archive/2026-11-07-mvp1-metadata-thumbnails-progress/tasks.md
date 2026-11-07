# Tasks

## 1. Núcleo

- [x] 1.1 `storage.PresignGet` e `storage.ThumbnailKey`, com teste de leitura parcial no RustFS
- [x] 1.2 Política pública inclui `assets/*/thumbnails/*`
- [x] 1.3 Migração `0008`: um `technical_metadata` por asset
- [x] 1.4 `store`: salvar/ler metadados, `SetJobProgress`, listagem com miniatura e resumo

## 2. Worker

- [x] 2.1 `ffmpeg.Probe` e `Summarize`, com nome substituindo a URL
- [x] 2.2 `ffmpeg.Run` com progresso (`-progress pipe:1`) e redação de segredos
- [x] 2.3 Handlers `ExtractMetadata`, `GenerateThumbnail` e `TranscodeHLS` em `jobs.Media`; stubs removidos
- [x] 2.4 Progresso com limite de frequência no `Runner`
- [x] 2.5 Testes com banco, storage e FFmpeg reais

## 3. API e dashboard

- [x] 3.1 `thumbnailUrl` e `metadata` na lista e no detalhe; `GET /assets/{id}/metadata`
- [x] 3.2 Lista com miniatura, duração e resolução
- [x] 3.3 Página do vídeo com capa, progresso da conversão e metadados

## 4. Documentação e verificação

- [x] 4.1 ADRs 0013 e 0014, `CONTEXT.md`, README
- [x] 4.2 Teste no navegador com vídeo de 90 s em 1080p e vídeo de 2 s
