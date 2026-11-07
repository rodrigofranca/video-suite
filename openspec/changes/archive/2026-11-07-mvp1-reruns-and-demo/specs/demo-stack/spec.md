# Spec Delta

## Purpose
Define como subir a plataforma inteira em Docker, sem Go, Node.js ou FFmpeg instalados, para demonstrações e avaliação.

## ADDED Requirements

### Requirement: Um comando para tudo
`make demo` SHALL construir as imagens da API, do worker e do dashboard e subir infraestrutura, migrações, API, worker e dashboard, terminando só quando todos estiverem prontos.

#### Scenario: Primeira execução
- **WHEN** o desenvolvedor executa `make demo` numa máquina só com Docker
- **THEN** o dashboard SHALL responder em `localhost:5173`, a API em `localhost:8080/healthz` e o banco SHALL estar migrado

### Requirement: Imagens
A imagem do worker SHALL incluir `ffmpeg` e `ffprobe` com libx264 e aac. As imagens da API e do worker SHALL rodar com usuário sem privilégios. O dashboard SHALL ser servido como arquivos estáticos, abrindo qualquer rota do app diretamente.

#### Scenario: Rota interna aberta direto
- **WHEN** o navegador abre `localhost:5173/assets/<id>` sem passar pela página inicial
- **THEN** o servidor SHALL responder com o app, que carrega a página do vídeo

### Requirement: Vídeo de exemplo
`make sample` SHALL criar `samples/sample.mp4` (30 segundos, 720p, com som) usando o FFmpeg da imagem do worker. A pasta `samples/` SHALL ficar fora do Git.

#### Scenario: Gerar o exemplo
- **WHEN** o desenvolvedor executa `make sample`
- **THEN** `samples/sample.mp4` SHALL existir e ser aceito pelo upload
