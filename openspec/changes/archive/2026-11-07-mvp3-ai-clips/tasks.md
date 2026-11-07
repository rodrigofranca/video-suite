# Tasks

## 1. Contratos e dados

- [x] 1.1 Tipos `suggest-clips` (pipeline, `jobs:ai`) e `export-clip` (sob demanda, `jobs`), formatos de corte, arquivo de contratos
- [x] 1.2 Migração `0011_create_clips`; `ReplaceClips`, `ListClips`, `GetClip`; `storage.ClipKey`; `PresignDownload`

## 2. Worker de IA

- [x] 2.1 `answers.py` compartilhado entre análise e cortes
- [x] 2.2 Job `suggest-clips` com encaixe nos trechos, limites de duração, sobreposição e nota
- [x] 2.3 Testes com o servidor falso; o servidor falso também sugere cortes para testes manuais

## 3. Worker Go

- [x] 3.1 Pacote `captions`: WebVTT, recorte, divisão e ASS
- [x] 3.2 `ffmpeg.ClipArgs` para os três formatos; job `export-clip` com progresso, upload e artefato
- [x] 3.3 Testes com FFmpeg real, inclusive conferindo os pixels da legenda queimada
- [x] 3.4 Fontes DejaVu na imagem do worker

## 4. API e dashboard

- [x] 4.1 `GET /assets/{id}/clips`, `POST /assets/{id}/clips/{clipId}/exports`, `GET /clips/{id}/download`
- [x] 4.2 Seção **Clips**: prévia, legendas, exportação, progresso, vídeo e download; exportações fora da lista de jobs

## 5. Entrega

- [x] 5.1 ADR 0020, README, CONTEXT, roteiro de demonstração
- [x] 5.2 Teste de ponta a ponta com o modelo real e na pilha Docker com o provedor simulado
