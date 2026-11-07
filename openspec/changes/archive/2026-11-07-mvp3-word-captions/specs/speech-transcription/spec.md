# Spec Delta

## MODIFIED Requirements

### Requirement: Transcrição com Whisper
O job SHALL transcrever o áudio extraído com faster-whisper, usando o modelo de `WHISPER_MODEL` (padrão `base`), detectando o idioma quando a entrada não informa um, e SHALL gravar em `transcripts` o idioma, o texto, os trechos com início e fim, as palavras com início e fim, e o modelo usado. As palavras SHALL estar em ordem, sem espaços nas pontas, cada uma começando no máximo quando a anterior termina. Rodar de novo SHALL substituir a transcrição anterior.

#### Scenario: Fala em inglês
- **WHEN** o áudio contém a frase sintética "Hello and welcome to the ultra media platform..."
- **THEN** a transcrição SHALL ter idioma `en` e conter "welcome" e "platform"

#### Scenario: Sem áudio
- **WHEN** não existe áudio extraído para o asset
- **THEN** o job SHALL terminar com sucesso, com o log `no audio to transcribe`, e nenhuma transcrição SHALL ser gravada

#### Scenario: Palavras com tempo
- **WHEN** a fala sintética é transcrita
- **THEN** a transcrição SHALL ter pelo menos 8 palavras em ordem, entre elas "welcome", dentro do tempo dos trechos
