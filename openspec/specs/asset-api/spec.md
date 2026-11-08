# asset-api Specification

## Purpose
Define as rotas de leitura da API que o dashboard e outras ferramentas usam para listar assets e inspecionar jobs, logs, artefatos e playback.

## Requirements

### Requirement: Listagem de assets
`GET /assets` SHALL retornar `{"assets": [...]}` com os assets do mais novo para o mais antigo, aceitando o filtro opcional `status`. Cada asset SHALL trazer `thumbnailUrl` e `metadata` (o resumo técnico), ambos `null` enquanto não existem.

#### Scenario: Filtro por status
- **WHEN** `GET /assets?status=ready` é chamado
- **THEN** SHALL retornar apenas assets `ready`

#### Scenario: Status desconhecido
- **WHEN** `GET /assets?status=weird` é chamado
- **THEN** SHALL responder 400

#### Scenario: Miniatura e metadados na lista
- **WHEN** um asset já tem miniatura e metadados
- **THEN** o item SHALL trazer a URL pública da miniatura e o resumo com duração e resolução

### Requirement: Detalhe do asset
`GET /assets/{id}` SHALL retornar o asset com `source`, `jobs` (na ordem do pipeline), `artifacts`, `playback` (que é `null` enquanto o asset não está `ready`), `thumbnailUrl`, `metadata` e `subtitles` (lista com `language` e `url`, a mais recente por idioma).

#### Scenario: Asset em processamento
- **WHEN** o asset está `processing`
- **THEN** a resposta SHALL trazer os jobs com status e `playback: null`

#### Scenario: ID inexistente ou malformado
- **WHEN** o ID não existe ou não é um UUID
- **THEN** SHALL responder 404 com `{"error": "not found"}`

#### Scenario: Antes dos metadados
- **WHEN** o job de metadata ainda não terminou
- **THEN** `thumbnailUrl` e `metadata` SHALL ser `null`

#### Scenario: Antes da transcrição
- **WHEN** a transcrição ainda não terminou
- **THEN** `subtitles` SHALL ser uma lista vazia

### Requirement: Informações de playback
`GET /assets/{id}/playback` SHALL retornar `{"type": "hls", "manifestUrl"}` para um asset `ready`, com a URL pública da playlist montada a partir de `S3_PUBLIC_URL`.

#### Scenario: Asset ainda não pronto
- **WHEN** o asset não está `ready`
- **THEN** SHALL responder 409 com o status atual

### Requirement: Jobs, logs e artefatos
A API SHALL expor `GET /assets/{id}/jobs`, `GET /assets/{id}/artifacts`, `GET /jobs/{id}` e `GET /jobs/{id}/logs`, com os logs em ordem cronológica.

#### Scenario: Logs de um job
- **WHEN** `GET /jobs/{id}/logs` é chamado para um job que já rodou
- **THEN** SHALL retornar `{"logs": [...]}` com nível, mensagem e horário de cada linha

### Requirement: CORS para o dashboard
A API SHALL responder com `Access-Control-Allow-Origin` apenas para a origem em `DASHBOARD_ORIGIN` (padrão `http://localhost:5173`) e SHALL responder preflights `OPTIONS` com 204.

#### Scenario: Origem não autorizada
- **WHEN** uma requisição chega com outra `Origin`
- **THEN** a resposta SHALL vir sem `Access-Control-Allow-Origin`

### Requirement: Metadados completos
`GET /assets/{id}/metadata` SHALL retornar o documento gravado pelo job de metadata, com `summary` e a saída completa do ffprobe.

#### Scenario: Metadados ainda não extraídos
- **WHEN** o job de metadata não terminou com sucesso
- **THEN** SHALL responder 404 com `{"error": "metadata not extracted yet"}`

### Requirement: Transcrição
`GET /assets/{id}/transcript` SHALL retornar `language`, `model`, `text`, `segments` (com `start`, `end` e `text`) e `createdAt`.

#### Scenario: Ainda não transcrito
- **WHEN** não há transcrição para o asset
- **THEN** SHALL responder 404 com `{"error": "transcript not available yet"}`

### Requirement: Análise da transcrição
`GET /assets/{id}/analysis` SHALL retornar `language`, `model`, `summary`, `tags` (lista), `chapters` (lista com `start` em segundos e `title`) e `createdAt`. Sem tags ou capítulos, os campos SHALL ser listas vazias, nunca `null`.

#### Scenario: Ainda não analisado
- **WHEN** não há análise para o asset
- **THEN** SHALL responder 404 com `{"error": "analysis not available yet"}`

#### Scenario: Análise refeita
- **WHEN** o job de análise roda de novo
- **THEN** a rota SHALL retornar só a análise mais recente

### Requirement: Cortes do asset
`GET /assets/{id}/clips` SHALL retornar `{"clips": [...], "wordTimings": <bool>}`, do melhor corte ao pior, cada um com `id`, `start`, `end`, `duration`, `title`, `reason`, `score`, `rank`, `model`, `faceCoverage` e `exports`: por formato, a exportação mais recente (`status`, `progress`, `error`) e, com arquivo, `ready`, `captions`, `captionStyle`, `framing`, `width`, `height` e `sizeBytes`. `wordTimings` SHALL dizer se há palavras com tempo.

#### Scenario: Exportação em andamento
- **WHEN** um corte tem uma exportação `pending` em `vertical`
- **THEN** o corte SHALL trazer essa exportação com `status: "pending"` e `ready: false`

#### Scenario: Transcrição antiga
- **WHEN** a transcrição do asset não tem palavras
- **THEN** `wordTimings` SHALL ser `false`

#### Scenario: Antes e depois do rastreio de rostos
- **WHEN** o corte ainda não tem `face_track`, e depois ganha um com `coverage` 0,85
- **THEN** `faceCoverage` SHALL ser `null`, e depois `0.85`

### Requirement: Pedir uma exportação
`POST /assets/{id}/clips/{clipId}/exports` com `{"format", "captions", "captionStyle"}` SHALL criar e enfileirar um job `export-clip` e responder 202 com o job. `captions` SHALL valer `true` e `captionStyle` SHALL valer `highlight` quando ausentes; os estilos são `highlight`, `box`, `karaoke`, `one-word` e `lines`. Formato ou estilo desconhecido SHALL responder 400; corte inexistente ou de outro asset, 404; e algum job do asset pendente ou rodando, 409.

#### Scenario: Pipeline ainda rodando
- **WHEN** a exportação é pedida enquanto o asset tem jobs pendentes
- **THEN** a API SHALL responder 409

#### Scenario: Estilo desconhecido
- **WHEN** o pedido traz `"captionStyle": "comic-sans"`
- **THEN** a API SHALL responder 400

### Requirement: Download de um corte
`GET /clips/{id}/download?format=...` SHALL redirecionar (302) para um link assinado de 15 minutos do arquivo exportado, que salva o arquivo como `<video>-clip-<n>-<formato>.mp4`; com `inline=1`, o link SHALL tocar no navegador. Sem arquivo nesse formato, SHALL responder 404.

#### Scenario: Nome do arquivo
- **WHEN** o corte 1 do vídeo `My Talk (final).mp4` é baixado em `vertical`
- **THEN** o arquivo SHALL se chamar `My-Talk-final-clip-1-vertical.mp4`

### Requirement: Rosto e falante nos cortes
Cada corte de `GET /assets/{id}/clips` SHALL ter também `faceFollows` (`speaker` ou `face`) e `facePeople` (pessoas juntas na tela), nulos antes do rastreio. Caminhos gravados antes de seguir quem fala SHALL aparecer como `face`, com `facePeople` nulo.

#### Scenario: Corte com duas pessoas
- **WHEN** o caminho do corte tem `follows: "speaker"` e `people: 2`
- **THEN** o corte SHALL trazer `faceFollows: "speaker"` e `facePeople: 2`

### Requirement: Edição nos cortes da API
Cada corte de `GET /assets/{id}/clips` SHALL ter `editedAt` (nulo se nunca editado), e cada exportação com arquivo SHALL ter `outdated: true` quando o arquivo foi cortado com outro início ou fim. O CORS SHALL permitir `PATCH`.

#### Scenario: Corte editado
- **WHEN** o título de um corte é mudado
- **THEN** a lista SHALL trazer o corte com `editedAt` preenchido

### Requirement: Opções de legenda na API
`POST /assets/{id}/clips/{clipId}/exports` SHALL aceitar `captionPosition`, `captionColor` e `captionCaps`, opcionais, respondendo 400 para valores fora das listas, e SHALL repassá-los ao job. Cada exportação pronta com legenda SHALL trazer `captionPosition`, `captionColor` e `captionCaps` usados.

#### Scenario: Cor desconhecida
- **WHEN** a exportação é pedida com `"captionColor": "red"`
- **THEN** a API SHALL responder 400 listando as cores

#### Scenario: Opções de volta
- **WHEN** uma exportação em `box` com `top`, `green` e maiúsculas termina
- **THEN** a lista de cortes SHALL trazer essa exportação com `captionPosition: "top"`, `captionColor: "green"` e `captionCaps: true`
