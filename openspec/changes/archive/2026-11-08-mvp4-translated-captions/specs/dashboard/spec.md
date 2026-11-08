# Spec Delta

## ADDED Requirements

### Requirement: Idioma das legendas no dashboard
As opções da legenda de um corte SHALL ter **Language**, com o idioma da transcrição (original) e os outros de `CaptionLanguages`. Num idioma ainda sem tradução, o painel SHALL avisar que a primeira exportação traduz a transcrição com o modelo. O arquivo pronto SHALL dizer o idioma da legenda traduzida, e o player SHALL oferecer a legenda traduzida como "(translated)", depois da original.

#### Scenario: Corte em espanhol
- **WHEN** o usuário escolhe **Spanish** e exporta **9:16 crop** num vídeo em inglês
- **THEN** o arquivo SHALL dizer "Spanish" e o player SHALL ter "Spanish (translated)"

### Requirement: Tradução em andamento no dashboard
Enquanto a tradução de uma exportação roda, o formato SHALL mostrar "Translating into …". A lista de jobs SHALL mostrar a tradução como "Translation · idioma", sem o botão de rodar de novo.

#### Scenario: Esperando a tradução
- **WHEN** a exportação em espanhol está pendente e a tradução roda
- **THEN** o formato SHALL mostrar "Translating into Spanish…"
