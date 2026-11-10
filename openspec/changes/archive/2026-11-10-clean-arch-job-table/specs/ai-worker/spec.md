## ADDED Requirements

### Requirement: Um handler por tipo de job
O worker de IA SHALL ter um handler para cada tipo de job que a tabela de tipos dá ao worker `ai`, e nenhum outro, e SHALL recusar partir quando isso não vale. Os jobs que usam o modelo de linguagem SHALL conferir o tamanho de todos os seus prompts antes da primeira chamada.

#### Scenario: Partida
- **WHEN** o worker de IA parte
- **THEN** os handlers SHALL ser exatamente `transcribe-audio`, `analyze-transcript`, `suggest-clips`, `track-faces` e `translate-transcript`
