# Spec Delta

## ADDED Requirements

### Requirement: Job de tradução
O pacote SHALL definir `translate-transcript`, roteado para `jobs:ai`, com `TranslateTranscriptInput` (`asset_id`, `language`); `CaptionLanguages` (`en`, `es`, `pt`, `fr`, `de`, `it`); e `caption_language` opcional em `ExportClipInput`. O arquivo de contratos compartilhado SHALL trazer `captionLanguages`.

#### Scenario: Exportação na língua falada
- **WHEN** um `ExportClipInput` sem idioma é codificado
- **THEN** o JSON SHALL não ter `caption_language`

### Requirement: Job que espera outro
O pacote SHALL definir `After`, com o job que pode esperar por um job fora do pipeline: `translate-transcript` é seguido por `export-clip`. O worker que termina um job SHALL pôr na fila o job pendente do tipo seguinte, pelo pipeline ou por `After`. O arquivo de contratos compartilhado SHALL trazer `after`.

#### Scenario: Tradução termina
- **WHEN** um `translate-transcript` termina, com sucesso ou não, e há um `export-clip` pendente do vídeo
- **THEN** o worker de IA SHALL pôr a exportação em `jobs`
