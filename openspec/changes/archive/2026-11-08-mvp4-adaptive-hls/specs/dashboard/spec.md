# Spec Delta

## ADDED Requirements

### Requirement: Qualidade do vídeo no dashboard
Com mais de uma qualidade no stream e `hls.js`, o player SHALL mostrar **Quality**, com **Auto** (e a qualidade em uso) e cada qualidade, da melhor para a pior; escolher uma SHALL fixá-la, e **Auto** SHALL voltar à escolha pela conexão. Os detalhes do vídeo SHALL listar as qualidades.

#### Scenario: Fixar 360p
- **WHEN** o usuário escolhe **360p** num vídeo 1080p
- **THEN** o vídeo SHALL passar a tocar com 360 de altura
