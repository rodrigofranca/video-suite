# Spec Delta

## Purpose
Define como ajustar um corte sugerido antes de exportar: início, fim e título, e o que muda depois.

## ADDED Requirements

### Requirement: Editar um corte
`PATCH /assets/{id}/clips/{clipId}` com qualquer um de `start`, `end` (segundos desde o início do vídeo, arredondados ao milésimo) e `title` SHALL mudar o corte no lugar, marcar `edited_at` e responder 200 com o corte como na lista. Espaços repetidos no título SHALL virar um. Sem nenhum campo, com campo desconhecido ou JSON inválido, SHALL responder 400; corte inexistente ou de outro asset, 404.

#### Scenario: Só o título
- **WHEN** o pedido é `{"title": "  The  hook\n"}`
- **THEN** o título SHALL ficar `The hook`, `editedAt` SHALL ser preenchido e nenhum job SHALL ser criado

### Requirement: Limites da edição
O início SHALL ser pelo menos 0 e antes do fim, o fim SHALL não passar da duração do vídeo nos metadados, o corte SHALL durar de 3 s a 3 minutos e o título SHALL ter de 1 a 120 caracteres; fora disso, a API SHALL responder 400 dizendo o motivo. Enquanto algum job do asset estiver pendente ou rodando, SHALL responder 409.

#### Scenario: Corte curto demais
- **WHEN** o pedido deixa o corte com 1,5 s
- **THEN** a API SHALL responder 400 e o corte SHALL continuar como estava

#### Scenario: Rastreio rodando
- **WHEN** o corte é editado enquanto um `track-faces` do asset está pendente
- **THEN** a API SHALL responder 409

### Requirement: Rastreio depois da edição
Quando início ou fim mudam, a mesma transação SHALL limpar `face_track` do corte e criar um `track-faces` `pending` com `clip_id`, e a API SHALL enfileirá-lo em `jobs:ai`. Se o enfileiramento falhar, a edição SHALL valer e o job SHALL ficar `failed` com o motivo.

#### Scenario: Novo intervalo
- **WHEN** o corte passa de 12–47,5 s para 10,25–40 s
- **THEN** `faceCoverage` SHALL ficar nulo e SHALL haver um `track-faces` enfileirado com o `clip_id` do corte

### Requirement: Exportações antigas
Uma exportação cujo artefato tem início ou fim diferentes dos atuais do corte SHALL aparecer com `outdated: true`; o arquivo SHALL continuar disponível até uma nova exportação.

#### Scenario: Exportado antes da edição
- **WHEN** o corte foi exportado em `vertical` e depois teve o início mudado
- **THEN** a exportação `vertical` SHALL ter `outdated: true` e continuar `ready`
