# Tasks

## 1. Contratos e pipeline

- [x] 1.1 Tipos `extract-audio` e `transcribe-audio`, entradas, `Pipeline` com cinco passos
- [x] 1.2 `StreamFor` e `Message.Stream`; API e worker enfileiram pelo stream do job
- [x] 1.3 `testdata/contracts.json` com teste Go (`-update`)
- [x] 1.4 Encadeamento continua depois do HLS

## 2. Áudio e transcrições

- [x] 2.1 Handler `extract-audio` (FLAC 16 kHz mono, vídeo sem áudio não é erro)
- [x] 2.2 Migração `0009_create_transcripts`, `store.SaveTranscript`/`GetTranscript`
- [x] 2.3 Política pública inclui `assets/*/subtitles/*`

## 3. Worker de IA

- [x] 3.1 Projeto `uv` com faster-whisper, redis, psycopg e boto3
- [x] 3.2 Contratos, store, storage, runner e consumer espelhando o Go
- [x] 3.3 Handler `transcribe-audio` com WebVTT e transcrição no banco
- [x] 3.4 Testes: contratos contra o arquivo do Go, legendas, ciclo de vida, fila e transcrição de fala sintética

## 4. API e dashboard

- [x] 4.1 `GET /assets/{id}/transcript` e `subtitles` no detalhe
- [x] 4.2 Legendas no player, transcrição clicável com trecho atual destacado

## 5. Entrega

- [x] 5.1 Imagem `ai-worker`, serviço e volume no Compose; targets do Makefile
- [x] 5.2 ADRs 0016 e 0017, README, `CONTEXT.md`, roteiro de demo
- [x] 5.3 Teste no navegador com fala sintética, local e na pilha Docker
