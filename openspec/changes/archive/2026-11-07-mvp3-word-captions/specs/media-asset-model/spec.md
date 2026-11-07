# Spec Delta

## ADDED Requirements

### Requirement: Palavras da transcrição
A migração `0012` SHALL adicionar a `transcripts` a coluna `words` (JSONB, `[]` por padrão), com as palavras e seus tempos.

#### Scenario: Transcrição antiga
- **WHEN** a migração roda num banco com transcrições
- **THEN** elas SHALL ficar com `words` igual a `[]`
