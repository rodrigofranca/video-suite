# Spec Delta

## ADDED Requirements

### Requirement: Listar uma pasta
O cliente SHALL listar as chaves dos objetos sob um prefixo, página a página.

#### Scenario: Arquivos antigos do HLS
- **WHEN** o job lista `assets/{id}/hls/` depois de enviar uma nova codificação
- **THEN** SHALL receber também os arquivos de codificações anteriores, para apagá-los
