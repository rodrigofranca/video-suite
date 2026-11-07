# Spec Delta

## MODIFIED Requirements

### Requirement: Makefile como orquestrador
O repositório SHALL incluir `Makefile` na raiz com targets documentados para as operações essenciais de desenvolvimento, carregando `.env` quando o arquivo existir. `make down` SHALL parar também a pilha de demo.

#### Scenario: Targets obrigatórios presentes
- **WHEN** o desenvolvedor executa `make help` ou `make`
- **THEN** os targets `up`, `down`, `demo`, `demo-logs`, `sample`, `migrate`, `migrate-down`, `dev-api`, `dev-worker`, `dev-dashboard`, `build` e `test` SHALL estar listados

#### Scenario: Migrate executa migrações
- **WHEN** o desenvolvedor executa `make migrate` com infraestrutura rodando
- **THEN** todas as migrações pendentes SHALL ser aplicadas ao banco de dados sem exigir a CLI `migrate` instalada globalmente

#### Scenario: Variável ausente gera mensagem clara
- **WHEN** um target que precisa do banco é executado sem `DATABASE_URL` definida
- **THEN** o Makefile SHALL avisar que a variável não está definida e sugerir `cp .env.example .env`
