# Proposal

## Why

Pedido em 2026-11-08: adicionar uma vinheta de abertura e/ou encerramento aos cortes, e o conceito de presets, que juntam a vinheta, o idioma e as outras escolhas de exportação (no futuro, a marca d'água).

## What Changes

- Uma biblioteca de vinhetas: MP4 curtos enviados uma vez, privados no storage, que podem ser tocados e apagados.
- Presets: nome, escolhas de legenda (ligada, estilo, posição, cor, maiúsculas, idioma), abertura e encerramento, cada um com uma versão para 16:9 e outra para 9:16.
- A exportação aceita `presetId`: o preset traz as vinhetas e as escolhas de legenda que o pedido deixa de fora.
- O worker junta abertura, corte e encerramento num só comando FFmpeg; uma vinheta de outro formato entra inteira sobre fundo desfocado.
- O painel ganha a página **Presets** e o seletor **Preset** em cada corte.

## Capabilities

### New Capabilities

- `export-presets`: biblioteca de vinhetas e presets na API.

### Modified Capabilities

- `clip-export`: vinhetas na exportação.
- `job-contracts`: vinhetas e preset no `ExportClipInput`.
- `dashboard`: página de presets e escolha do preset.

## Impact

- Código: migração 0016, `packages/media-core` (`Vignette`, `Preset`, store, `VignetteKey`), `packages/job-contracts`, `apps/worker` (`ffmpeg/clip.go`, `jobs/clip.go`), `apps/api` (`presets.go`, `clips.go`), `apps/dashboard` (página `/presets`, `Clips`).
- Docs: ADR 0030, README, CONTEXT, docs/README, demo e backlog.
