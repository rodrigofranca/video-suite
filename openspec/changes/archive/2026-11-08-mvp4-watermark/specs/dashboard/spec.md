# Spec Delta

## ADDED Requirements

### Requirement: Marca d'água no dashboard
A página **Presets** SHALL ter a seção **Watermarks**, para enviar e apagar imagens, com miniatura e tamanho, e o formulário do preset SHALL ter **Watermark**: imagem, canto, tamanho e opacidade, com uma prévia num quadro 9:16 e num 16:9. Com um preset de marca d'água, o aviso do corte SHALL dizer qual imagem entra, e o arquivo pronto SHALL mostrar "watermark".

#### Scenario: Prévia
- **WHEN** o usuário escolhe uma imagem, o canto de baixo à esquerda e 25%
- **THEN** a prévia SHALL mostrar a imagem nesse canto nos dois quadros
