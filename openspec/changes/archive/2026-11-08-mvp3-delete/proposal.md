# Proposal

## Why

Pedido em 2026-11-08, primeira parte de "pronto para produção": nada podia ser apagado. Vídeos de teste, cortes ruins e todos os arquivos ficavam para sempre.

## What Changes

- `DELETE /assets/{id}`: apaga o vídeo, tudo o que foi registrado sobre ele e todos os arquivos sob `assets/{id}/`.
- `DELETE /assets/{id}/clips/{clipId}`: apaga o corte e os arquivos exportados.
- Os dois esperam o vídeo ficar ocioso (409); um upload em andamento não pode ser apagado, e um parado há mais de uma hora conta como abandonado.
- `DeletePrefix` no storage, que só aceita a pasta de um asset ou uma pasta dentro dela.
- Painel: **Delete video** no fim da página e **🗑 Delete** em cada corte, com confirmação.

## Capabilities

### New Capabilities

- `asset-deletion`: apagar vídeos e cortes.

### Modified Capabilities

- `object-storage`: apagar uma pasta.
- `dashboard`: botões de apagar.

## Impact

- Código: `packages/media-core` (`storage.DeletePrefix`, `AssetPrefix`, `ClipPrefix`, `store.DeleteAssetIfIdle`, `DeleteClipIfIdle`), `apps/api` (`delete.go`, CORS com DELETE) e `apps/dashboard`.
- Docs: ADR 0027, README, CONTEXT, roteiro da demo e backlog.
