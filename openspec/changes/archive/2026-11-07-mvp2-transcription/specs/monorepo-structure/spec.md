# Spec Delta

## MODIFIED Requirements

### Requirement: Estrutura de diretórios do monorepo
O repositório SHALL seguir estrutura `apps/` para aplicações e `packages/` para pacotes compartilhados, sem tooling de monorepo (Turborepo/Nx) no MVP. O worker Python fica em `apps/ai-worker`, gerenciado com `uv`.

#### Scenario: Estrutura de diretórios presente
- **WHEN** o repositório é clonado
- **THEN** os diretórios `apps/api`, `apps/worker`, `apps/ai-worker`, `apps/dashboard`, `packages/job-contracts` e `packages/media-core` SHALL existir

#### Scenario: Go Workspace configurado
- **WHEN** o desenvolvedor executa `go work sync` na raiz
- **THEN** os módulos `apps/api`, `apps/worker`, `packages/job-contracts` e `packages/media-core` SHALL estar referenciados em `go.work`

#### Scenario: Código-fonte nunca é ignorado pelo Git
- **WHEN** `git check-ignore` é executado sobre arquivos `.go` ou `go.mod` em `apps/` ou `packages/`
- **THEN** nenhum desses arquivos SHALL ser ignorado; apenas os binários `/apps/api/api` e `/apps/worker/worker` SHALL ser ignorados

### Requirement: Makefile como orquestrador
O repositório SHALL incluir `Makefile` na raiz com targets documentados para as operações essenciais de desenvolvimento, carregando `.env` quando o arquivo existir. `make down` SHALL parar também a pilha de demo.

#### Scenario: Targets obrigatórios presentes
- **WHEN** o desenvolvedor executa `make help` ou `make`
- **THEN** os targets `up`, `down`, `demo`, `demo-logs`, `sample`, `sample-speech`, `migrate`, `migrate-down`, `dev-api`, `dev-worker`, `dev-ai-worker`, `dev-dashboard`, `build`, `test`, `test-go` e `test-ai` SHALL estar listados

#### Scenario: Migrate executa migrações
- **WHEN** o desenvolvedor executa `make migrate` com infraestrutura rodando
- **THEN** todas as migrações pendentes SHALL ser aplicadas ao banco de dados sem exigir a CLI `migrate` instalada globalmente

#### Scenario: Variável ausente gera mensagem clara
- **WHEN** um target que precisa do banco é executado sem `DATABASE_URL` definida
- **THEN** o Makefile SHALL avisar que a variável não está definida e sugerir `cp .env.example .env`
