# Spec Delta

## ADDED Requirements

### Requirement: Exportar com legendas traduzidas
`POST /assets/{id}/clips/{clipId}/exports` SHALL aceitar `captionLanguage`, um de `CaptionLanguages` (outro valor dá 400). Com legendas e um idioma diferente do da transcrição sem tradução nesse idioma, a API SHALL criar, na mesma transação, um `translate-transcript` na fila e o `export-clip` pendente, e responder 202 com a exportação; sem transcrição SHALL responder 409. Com a tradução pronta, com o idioma da transcrição ou sem legendas, SHALL criar só a exportação.

#### Scenario: Primeiro corte em inglês
- **WHEN** a transcrição está em português, sem traduções, e a exportação pede `en`
- **THEN** a fila SHALL receber só o `translate-transcript`, e o `export-clip` SHALL ficar pendente com `caption_language` `en`

### Requirement: Traduções nos cortes e no asset
`GET /assets/{id}/clips` SHALL trazer `transcriptLanguage` e `translations`, e cada exportação SHALL trazer `captionLanguage` (do arquivo pronto, ou o pedido) e `translatedFrom`. O download de um arquivo com legenda traduzida SHALL ter o idioma no nome (`-vertical-en.mp4`). As legendas do asset SHALL vir com a original primeiro e depois as traduções da transcrição atual, com `translatedFrom`.

#### Scenario: Transcrição refeita
- **WHEN** a transcrição de um vídeo traduzido é refeita
- **THEN** o asset SHALL listar só a legenda original
