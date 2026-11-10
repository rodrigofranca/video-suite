## ADDED Requirements

### Requirement: Busca no dashboard
O menu SHALL ter **Search**, e a página **Videos** SHALL ter uma caixa "Search what was said…". A página de busca SHALL mostrar dicas antes da primeira busca. Os resultados SHALL ter, por vídeo, a miniatura, o nome, a data, o idioma e os momentos com o tempo e as palavras marcadas, e quantos momentos a mais o vídeo tem. Sem resultado, SHALL dizer em quantos vídeos com transcrição procurou. **More videos** SHALL trazer a página seguinte.

#### Scenario: Nada encontrado
- **WHEN** a busca não acha nada em 32 vídeos com transcrição
- **THEN** o painel SHALL dizer "Nothing found for “…” in 32 videos with a transcript."
