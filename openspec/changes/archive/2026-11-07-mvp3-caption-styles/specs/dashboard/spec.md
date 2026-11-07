# Spec Delta

## MODIFIED Requirements

### Requirement: Escolha das legendas no dashboard
Cada corte SHALL oferecer **Captions**: **Highlighted word** (padrão, quando a transcrição tem palavras), **Word box**, **Karaoke**, **One word at a time**, **Lines** (quando o vídeo tem legendas) ou **Off**. Os estilos de palavras SHALL ficar desativados sem palavras na transcrição; nesse caso, se o vídeo tem legendas, a seção SHALL dizer uma vez que basta rodar a transcrição de novo.

#### Scenario: Vídeo transcrito antes das palavras
- **WHEN** o usuário abre um vídeo transcrito sem palavras e roda **Transcription** de novo
- **THEN** antes os estilos de palavras SHALL estar desativados, com o aviso, e depois **Highlighted word** SHALL ser a escolha padrão, sem o aviso

#### Scenario: Exportar em karaokê
- **WHEN** o usuário escolhe **Karaoke** e clica em **9:16 blur**
- **THEN** a exportação SHALL terminar mostrando o estilo **Karaoke**
