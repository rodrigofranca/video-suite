# Spec Delta

## ADDED Requirements

### Requirement: Resumo, tags e capítulos no dashboard
A página do vídeo SHALL mostrar uma seção **Summary** com o resumo, as tags, os capítulos com horário e o modelo que os escreveu. Clicar num capítulo SHALL levar o player ao início dele, e o capítulo em reprodução SHALL ficar destacado. Enquanto a análise roda, a seção SHALL dizer que o resumo está sendo escrito; se a análise falhou ou o vídeo não tem fala, SHALL dizer isso.

#### Scenario: Clique num capítulo
- **WHEN** o usuário clica no capítulo que começa em 0:49
- **THEN** o player SHALL ir para cerca de 49 segundos e esse capítulo SHALL ficar destacado

#### Scenario: Análise com falha
- **WHEN** o job de resumo falhou
- **THEN** a seção SHALL dizer que o resumo não pôde ser escrito e apontar para o job, que oferece **Run again**
