# Spec Delta

## MODIFIED Requirements

### Requirement: Imagens
A imagem do worker SHALL incluir `ffmpeg` e `ffprobe` com libx264, aac e libass, e as fontes DejaVu usadas nas legendas queimadas. As imagens da API e do worker SHALL rodar com usuário sem privilégios. O dashboard SHALL ser servido como arquivos estáticos, abrindo qualquer rota do app diretamente.

#### Scenario: Rota interna aberta direto
- **WHEN** o navegador abre `localhost:5173/assets/<id>` sem passar pela página inicial
- **THEN** o servidor SHALL responder com o app, que carrega a página do vídeo

#### Scenario: Corte com legendas no Docker
- **WHEN** um corte é exportado com legendas pela pilha de `make demo`
- **THEN** as legendas SHALL aparecer no vídeo
