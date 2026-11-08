# Proposal

## Why

Pedido em 2026-11-08: com duas ou mais pessoas na tela, o 9:16 com rosto seguia o maior rosto, que muitas vezes não é o de quem fala. Num podcast ou entrevista, a câmera deve ir para quem está falando. E sugerir os cortes de novo deixava os cortes novos sem rastreio.

## What Changes

- `track-faces` liga os rostos em trilhas, uma por pessoa. Com duas ou mais pessoas juntas e som, recorta cada rosto a 25 quadros por segundo e usa o Light-ASD (modelo audiovisual, MIT) para saber quem fala; a câmera vai para essa pessoa, com regras para não piscar.
- O modelo roda no ONNX Runtime, sem PyTorch: os pesos oficiais são baixados de um commit fixo, conferidos por SHA-256 e lidos por um leitor restrito.
- O caminho do corte ganha `follows` e `people`; a API devolve `faceFollows` e `facePeople`; o painel mostra `Follows who speaks · 2 people`.
- Rodar `suggest-clips` de novo cria também um `track-faces` pendente, que roda depois dele.

## Capabilities

### Modified Capabilities

- `face-tracking`: trilhas, quem fala e o modelo de falante.
- `job-rerun` e `job-contracts`: jobs refeitos juntos.
- `asset-api`: `faceFollows` e `facePeople`.
- `dashboard`: o que **9:16 face** segue.
- `ai-worker`: carrega o modelo de falante na inicialização.
- `demo-stack`: o modelo fica no volume dos modelos.

## Impact

- Código: `apps/ai-worker` (`speakers.py` novo, `faces.py`; `onnx` e `onnxruntime` como dependências diretas), `packages/job-contracts`, `packages/media-core/store`, `apps/api` e `apps/dashboard`.
- Docs: ADR 0023, README, CONTEXT, backlog e roteiro da demo.
