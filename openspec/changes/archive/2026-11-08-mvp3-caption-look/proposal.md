# Proposal

## Why

Pedido em 2026-11-07 e escolhido em 2026-11-08: além do estilo, poder escolher onde a legenda fica, a cor do destaque e se o texto sai em maiúsculas, a cada exportação.

## What Changes

- Posição: embaixo (padrão), no meio ou em cima.
- Cor do destaque: amarelo, verde, rosa, azul ou violeta, para a palavra destacada, a caixa e o karaokê; na caixa, a palavra fica preta ou branca, a que se lê melhor.
- Maiúsculas ligadas ou desligadas em qualquer estilo; sem escolha, só `one-word`.
- Contrato, API e artefato com as opções; o painel ganha os controles e mostra o que foi usado.

## Capabilities

### Modified Capabilities

- `clip-export`: aparência das legendas.
- `job-contracts`: opções de legenda em `ExportClipInput`.
- `asset-api`: opções no pedido e na exportação.
- `dashboard`: controles de posição, cor e maiúsculas.

## Impact

- Código: `apps/worker/internal/captions` (`Options`, `Style.With`), `jobs/clip.go`, `packages/job-contracts`, `apps/api` e `apps/dashboard`.
- Docs: ADR 0025, README, CONTEXT, roteiro da demo; o item saiu do backlog.
