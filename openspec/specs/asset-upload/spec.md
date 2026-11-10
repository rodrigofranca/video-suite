# asset-upload Specification

## Purpose
Define como um vídeo entra na plataforma: o endpoint de upload, as validações e a criação do asset com seu arquivo de origem e seus jobs.

## Requirements

### Requirement: Upload de MP4 em streaming
`POST /assets/upload` SHALL aceitar `multipart/form-data` com o campo `file` e enviar o conteúdo direto ao storage, sem guardar o arquivo inteiro em memória ou disco. A resposta de sucesso SHALL ser `201` com `{"assetId", "status": "processing"}`.

#### Scenario: Upload válido
- **WHEN** um arquivo `.mp4` válido é enviado
- **THEN** a API SHALL responder 201, o arquivo SHALL estar no storage em `assets/{assetId}/source/input.mp4` e o source SHALL registrar tamanho e SHA-256 corretos

#### Scenario: Requisição sem arquivo
- **WHEN** o corpo não é multipart ou não tem o campo `file`
- **THEN** a API SHALL responder 400

### Requirement: Validação do arquivo
A API SHALL aceitar apenas arquivos com extensão `.mp4` cujo conteúdo começa com a caixa `ftyp` do formato MP4, e SHALL recusar arquivos acima de 2 GB antes de gravar qualquer registro definitivo.

#### Scenario: Extensão ou conteúdo inválido
- **WHEN** o arquivo não termina em `.mp4` ou não começa com a caixa `ftyp`
- **THEN** a API SHALL responder 415 sem criar asset

#### Scenario: Arquivo grande demais com tamanho declarado
- **WHEN** o `Content-Length` excede o limite
- **THEN** a API SHALL responder 413 sem ler o corpo

#### Scenario: Arquivo grande demais sem tamanho declarado
- **WHEN** o corpo é enviado sem `Content-Length` e passa do limite durante o envio
- **THEN** a API SHALL responder 413, apagar o objeto parcial do storage e apagar o asset

### Requirement: Criação do asset e do pipeline
Após o arquivo ser gravado, a API SHALL registrar o source e um job `pending` por passo do pipeline, mudar o asset para `processing` e pôr apenas o primeiro job na fila pela outbox (ADR 0034), tudo numa única transação.

#### Scenario: Jobs criados e primeiro enfileirado
- **WHEN** o upload termina com sucesso
- **THEN** o asset SHALL ter os jobs `extract-metadata`, `generate-thumbnail`, `transcode-hls`, `extract-audio`, `transcribe-audio`, `analyze-transcript`, `suggest-clips` e `track-faces` em `pending`, nessa ordem, e só `extract-metadata` SHALL estar na fila

#### Scenario: Fila indisponível
- **WHEN** o Redis está fora do ar durante o upload
- **THEN** a API SHALL responder 201, e o primeiro job SHALL continuar `pending` até o relay conseguir mandá-lo

### Requirement: Chaves de storage geradas
O arquivo SHALL ser gravado numa chave derivada do ID do asset. O nome original do arquivo SHALL ser guardado apenas como `filename` (sem diretórios) e nunca usado para montar caminhos.

#### Scenario: Nome de arquivo com caminho
- **WHEN** o arquivo enviado se chama `../../etc/passwd.mp4`
- **THEN** o `filename` SHALL ser `passwd.mp4` e a chave SHALL continuar sendo `assets/{assetId}/source/input.mp4`
