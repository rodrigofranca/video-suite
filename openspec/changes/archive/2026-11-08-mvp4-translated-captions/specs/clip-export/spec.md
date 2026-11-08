# Spec Delta

## ADDED Requirements

### Requirement: Legendas em outro idioma
Com `caption_language`, o job SHALL queimar a legenda da tradução da transcrição nesse idioma, em qualquer estilo: as palavras da tradução nos estilos palavra a palavra e a legenda WebVTT traduzida em `lines`. Sem idioma, ou com o idioma da transcrição, SHALL usar a original, mesmo que uma legenda traduzida seja mais nova. O artefato `clip` SHALL registrar `language` e `translatedFrom`.

#### Scenario: Sem tradução
- **WHEN** a exportação pede francês e a transcrição não tem tradução em francês
- **THEN** o job SHALL falhar dizendo que não há tradução para `fr`, sem gerar arquivo

#### Scenario: Idioma fora da lista
- **WHEN** `caption_language` é `ja`
- **THEN** o job SHALL recusar a entrada
