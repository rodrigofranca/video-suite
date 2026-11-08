# Spec Delta

## Purpose
Define como apagar um vídeo ou um corte, com os registros e os arquivos guardados.

## ADDED Requirements

### Requirement: Apagar um vídeo
`DELETE /assets/{id}` SHALL apagar o asset, tudo o que o banco guarda dele (por cascata) e, depois, todos os arquivos sob `assets/{id}/`, respondendo 204. Se o storage falhar depois do banco, a resposta SHALL continuar 204 e o log SHALL dizer a pasta dos arquivos que sobraram. Um asset inexistente SHALL dar 404.

#### Scenario: Vídeo com cortes e arquivos
- **WHEN** um vídeo pronto, com um corte exportado, é apagado
- **THEN** `GET /assets/{id}` SHALL dar 404 e SHALL não sobrar nenhum arquivo sob `assets/{id}/`

### Requirement: Apagar um corte
`DELETE /assets/{id}/clips/{clipId}` SHALL apagar o corte, os registros dos arquivos exportados dele e a pasta `assets/{id}/clips/{clipId}/`, respondendo 204; os jobs de exportação SHALL continuar no histórico. Corte inexistente, de outro asset ou id inválido SHALL dar 404.

#### Scenario: Um corte entre dois
- **WHEN** um de dois cortes exportados é apagado
- **THEN** a lista SHALL ter só o outro, e só os arquivos do apagado SHALL sumir

### Requirement: Só com o vídeo ocioso
Os dois pedidos SHALL responder 409 enquanto algum job do asset estiver pendente ou rodando. Apagar o vídeo também SHALL responder 409 enquanto ele está em upload, exceto se o upload parou há mais de uma hora.

#### Scenario: Pipeline rodando
- **WHEN** o vídeo ou um corte é apagado enquanto o pipeline ainda tem jobs pendentes
- **THEN** a API SHALL responder 409 e nada SHALL ser apagado

#### Scenario: Upload abandonado
- **WHEN** um asset está `uploading` há duas horas
- **THEN** ele SHALL poder ser apagado
