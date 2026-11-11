# Proposal

## Why

Terceiro dos três itens que subiram no backlog em 2026-11-10: tirar as pausas e os vícios de fala com cortes secos, como no Reels e no TikTok, sem editor de vídeo.

## What Changes

- O plano dos cortes a partir do tempo das palavras: cada palavra com uma folga, pausas curtas mantidas, vícios de fala fora.
- Um estágio de exportação novo, `JumpCuts`, que recorta e junta imagem e som com `trim`, `atrim` e `concat`, depois da legenda e antes das vinhetas.
- `jumpCuts` na exportação e no preset (migração 0022), com 409 sem o tempo das palavras.
- No painel: **Jump cuts** em cada corte e no preset, e a nova duração no arquivo exportado.

## Capabilities

### New Capabilities

- `jump-cuts`: cortes secos numa exportação.

### Modified Capabilities

- `export-presets`: o preset guarda os cortes secos.
- `dashboard`: a opção no corte e no preset.

## Impact

- Código: migração 0022, `apps/worker` (`jumpcut`, `ffmpeg/stages.go`, `jobs/clip.go`), `packages/job-contracts`, `packages/media-core` (preset), `apps/api` (`app/export.go`, `app/presets.go`, `httpapi`), `apps/dashboard` (`ExportChoices`, `ClipCard`, `PresetForm`, `clips.ts`).
- Docs: ADR 0040, README, CONTEXT, docs/README, demo e backlog.
