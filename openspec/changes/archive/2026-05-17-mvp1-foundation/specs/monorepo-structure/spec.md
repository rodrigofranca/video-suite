## ADDED Requirements

### Requirement: Estrutura de diretórios do monorepo
O repositório SHALL seguir estrutura `apps/` para aplicações e `packages/` para pacotes compartilhados, sem tooling de monorepo (Turborepo/Nx) no MVP.

#### Scenario: Estrutura de diretórios presente
- **WHEN** o repositório é clonado
- **THEN** os diretórios `apps/api`, `apps/worker`, `apps/dashboard`, `packages/job-contracts` SHALL existir

#### Scenario: Go Workspace configurado
- **WHEN** o desenvolvedor executa `go work sync` na raiz
- **THEN** os módulos `apps/api`, `apps/worker` e `packages/job-contracts` SHALL estar referenciados em `go.work`

### Requirement: Docker Compose para infraestrutura local
O repositório SHALL incluir `docker-compose.yml` na raiz que sobe PostgreSQL, Redis e MinIO prontos para desenvolvimento.

#### Scenario: Infraestrutura sobe em um comando
- **WHEN** o desenvolvedor executa `make up`
- **THEN** PostgreSQL SHALL estar acessível em `localhost:5432`, Redis em `localhost:6379` e MinIO em `localhost:9000`

#### Scenario: Infraestrutura desce em um comando
- **WHEN** o desenvolvedor executa `make down`
- **THEN** todos os containers SHALL ser parados e removidos

### Requirement: Makefile como orquestrador
O repositório SHALL incluir `Makefile` na raiz com targets documentados para as operações essenciais de desenvolvimento.

#### Scenario: Targets obrigatórios presentes
- **WHEN** o desenvolvedor executa `make help` ou `make`
- **THEN** os targets `up`, `down`, `migrate`, `dev-api`, `dev-worker`, `dev-dashboard` SHALL estar listados

#### Scenario: Migrate executa migrações
- **WHEN** o desenvolvedor executa `make migrate` com infraestrutura rodando
- **THEN** todas as migrações pendentes SHALL ser aplicadas ao banco de dados
