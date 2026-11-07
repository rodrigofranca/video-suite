# Spec Delta

## ADDED Requirements

### Requirement: Transcrição e legendas no dashboard
A página do vídeo SHALL carregar as legendas do asset no player e mostrar uma seção **Transcript** com o idioma, o modelo e os trechos com horário. Clicar num trecho SHALL levar o player àquele ponto, e o trecho em reprodução SHALL ficar destacado. Enquanto a transcrição roda, a seção SHALL mostrar o progresso; se o vídeo não tem áudio ou a transcrição falhou, SHALL dizer isso.

#### Scenario: Clique num trecho
- **WHEN** o usuário clica no trecho que começa em 0:07
- **THEN** o player SHALL ir para cerca de 7 segundos e o trecho SHALL ficar destacado

#### Scenario: Legendas no player
- **WHEN** o vídeo tem legendas em inglês
- **THEN** o player SHALL ter uma faixa de legendas `English` carregada do storage
