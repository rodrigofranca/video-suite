# Proposal

## Why

Pedido em 2026-11-08, depois das vinhetas: a marca d'água, um logo do canal sobre o corte, guardada no preset.

## What Changes

- Uma biblioteca de imagens de marca d'água (PNG ou JPEG), privada, com o tamanho lido no envio.
- O preset ganha a imagem, o canto, o tamanho (porcentagem do lado menor do corte) e a opacidade.
- A exportação com preset põe a imagem sobre o corte, abaixo da legenda e fora das vinhetas, no mesmo comando FFmpeg.
- O painel ganha a seção **Watermarks** e a parte **Watermark** no formulário do preset, com prévia nos dois formatos.

## Capabilities

### Modified Capabilities

- `export-presets`: imagens de marca d'água e escolhas no preset.
- `clip-export`: a marca d'água na exportação.
- `job-contracts`: `watermark` no `ExportClipInput`.
- `dashboard`: marca d'água nos presets e nos cortes.

## Impact

- Código: migração 0017, `packages/media-core` (`Watermark`, store, `WatermarkKey`), `packages/job-contracts`, `apps/worker` (`ffmpeg/clip.go`, `jobs/clip.go`), `apps/api` (`watermarks.go`, `presets.go`, `clips.go`), `apps/dashboard` (`/presets`, `Clips`).
- Docs: ADR 0031, README, CONTEXT, docs/README, demo e backlog.
