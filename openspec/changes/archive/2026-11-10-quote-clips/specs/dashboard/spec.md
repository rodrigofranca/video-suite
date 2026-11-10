## ADDED Requirements

### Requirement: Cortes sem sugestão
A lista de cortes SHALL carregar quando a transcrição termina, mesmo que a sugestão de cortes tenha falhado, e as mensagens de falha e de "nenhum corte" SHALL lembrar que dá para fazer um corte de uma citação.

#### Scenario: Sugestão falhou
- **WHEN** a sugestão falha e o usuário faz um corte de citação
- **THEN** a seção **Clips** SHALL mostrar o corte
