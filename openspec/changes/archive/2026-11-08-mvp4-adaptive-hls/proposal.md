# Proposal

## Why

Pedido em 2026-11-08, segunda parte de "pronto para produção": o HLS saía numa qualidade só, a do original. Em conexões fracas, um vídeo 1080p travava. O player deve escolher a qualidade que a conexão aguenta.

## What Changes

- `transcode-hls` gera até quatro qualidades pelo lado menor (1080p, 720p, 480p, 360p), sem passar do original, num só comando FFmpeg com quadros-chave alinhados e teto de bitrate por qualidade.
- O playlist mestre fica em `hls/index.m3u8`, o endereço de antes; cada qualidade numa pasta.
- Depois do envio, o job apaga os arquivos de `hls/` que não escreveu (codificações anteriores).
- A API devolve `playback.renditions`; o painel mostra as qualidades e um seletor **Quality** no player.

## Capabilities

### Modified Capabilities

- `hls-transcoding`: várias qualidades, envio e limpeza.
- `asset-api`: qualidades no playback.
- `dashboard`: seletor de qualidade.
- `object-storage`: listar uma pasta.

## Impact

- Código: `apps/worker/internal/ffmpeg` (`Ladder`, `HLSArgs`), `jobs/hls.go`, `packages/media-core/storage` (`List`), `apps/api` e `apps/dashboard` (`HlsPlayer`).
- Docs: ADR 0028, README, CONTEXT, docs/README, demo e backlog.
