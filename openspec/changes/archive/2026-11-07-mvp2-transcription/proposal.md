# Proposal

## Why

Primeira entrega do Milestone 2 (AI metadata). Com o pipeline de vídeo completo, o próximo valor é entender o que é dito no vídeo: transcrição e legendas são a base de busca, capítulos, resumos e cortes nas próximas entregas. Decisões confirmadas em 2026-11-07: Whisper local num worker Python (ADR 0007) e, nesta entrega, só transcrição e legendas (resumo, tags e capítulos com LLM ficam para a segunda entrega).

## What Changes

- Dois jobs novos no fim do pipeline: `extract-audio` (worker Go, FFmpeg) e `transcribe-audio` (worker Python novo).
- `apps/ai-worker`: worker Python (faster-whisper) que lê o stream `jobs:ai` e segue o mesmo ciclo de vida do worker Go.
- Roteamento de jobs por stream (`jobcontracts.StreamFor`) e um arquivo de contratos que os testes Go e Python conferem.
- O encadeamento continua depois do HLS; o asset fica `ready` antes da transcrição.
- Migração `0009_create_transcripts`; `GET /assets/{id}/transcript`; `subtitles` no detalhe do asset.
- Legendas WebVTT públicas em `assets/{id}/subtitles/`; áudio privado em `assets/{id}/audio/`.
- Dashboard: legendas no player e transcrição clicável que leva o vídeo ao trecho.
- Docker: imagem e serviço `ai-worker` com volume para os modelos; `make dev-ai-worker`, `make test-ai`, `make sample-speech`.

## Capabilities

### New Capabilities

- `audio-extraction`: extração do áudio do vídeo para a transcrição.
- `speech-transcription`: transcrição da fala e geração de legendas.
- `ai-worker`: o processo Python que executa os jobs de IA.

### Modified Capabilities

- `job-contracts`: novos tipos, roteamento por stream e arquivo de contratos compartilhado.
- `processing-pipeline`: cinco passos e encadeamento depois do HLS.
- `asset-upload`: o upload cria os cinco jobs.
- `object-storage`: áudio privado e legendas públicas.
- `asset-api`: legendas no detalhe e rota da transcrição.
- `dashboard`: legendas e transcrição.
- `media-asset-model`: tabela de transcrições.
- `monorepo-structure`: `apps/ai-worker` e novos targets.
- `demo-stack`: serviço `ai-worker`.

## Impact

- Código: `apps/ai-worker` (novo), `apps/worker` (extract-audio, encadeamento), `apps/api`, `apps/dashboard`, `packages/job-contracts`, `packages/media-core`.
- Dependências: `faster-whisper` 1.2.1, `av` <19, `redis`, `psycopg`, `boto3`; imagem `python:3.13.17-slim-bookworm` e `uv` 0.13.5.
- Rede: o worker de IA baixa o modelo do Hugging Face na primeira inicialização.
- Docs: ADRs 0016 e 0017.
