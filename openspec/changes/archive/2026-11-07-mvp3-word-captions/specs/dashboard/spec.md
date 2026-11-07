# Spec Delta

## MODIFIED Requirements

### Requirement: Exportar cortes pelo dashboard
Cada corte SHALL ter um botão por formato (**Original**, **9:16 crop**, **9:16 blur**). Durante a exportação, o formato SHALL mostrar o progresso; ao terminar, o vídeo exportado, o link **Download** e o estilo das legendas. Os botões SHALL ficar desativados enquanto algum job do vídeo estiver pendente ou rodando, e os jobs de exportação SHALL não aparecer na lista de jobs do pipeline.

#### Scenario: Exportar em 9:16 com desfoque
- **WHEN** o usuário clica em **9:16 blur** no primeiro corte
- **THEN** a página SHALL mostrar o progresso e, ao terminar, o vídeo vertical e o link de download

## ADDED Requirements

### Requirement: Escolha das legendas no dashboard
Cada corte SHALL oferecer **Captions**: **Word by word** (padrão, quando a transcrição tem palavras), **Lines** (quando o vídeo tem legendas) ou **Off**. Se o vídeo tem legendas mas não palavras, a seção SHALL dizer uma vez que basta rodar a transcrição de novo.

#### Scenario: Vídeo transcrito antes das palavras
- **WHEN** o usuário abre um vídeo transcrito sem palavras e roda **Transcription** de novo
- **THEN** antes **Word by word** SHALL estar desativado, com o aviso, e depois SHALL ser a escolha padrão, sem o aviso
