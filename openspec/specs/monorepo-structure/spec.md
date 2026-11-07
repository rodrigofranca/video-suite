# monorepo-structure Specification

## Purpose
Define a organização do monorepo e o ambiente de desenvolvimento local: pastas, Go Workspace, Docker Compose e Makefile.

## Requirements

### Requirement: Estrutura de diretórios do monorepo
O repositório SHALL seguir estrutura `apps/` para aplicações e `packages/` para pacotes compartilhados, sem tooling de monorepo (Turborepo/Nx) no MVP.

#### Scenario: Estrutura de diretórios presente
- **WHEN** o repositório é clonado
- **THEN** os diretórios `apps/api`, `apps/worker`, `apps/dashboard`, `packages/job-contracts` e `packages/media-core` SHALL existir

#### Scenario: Go Workspace configurado
- **WHEN** o desenvolvedor executa `go work sync` na raiz
- **THEN** os módulos `apps/api`, `apps/worker`, `packages/job-contracts` e `packages/media-core` SHALL estar referenciados em `go.work`

#### Scenario: Código-fonte nunca é ignorado pelo Git
- **WHEN** `git check-ignore` é executado sobre arquivos `.go` ou `go.mod` em `apps/` ou `packages/`
- **THEN** nenhum desses arquivos SHALL ser ignorado; apenas os binários `/apps/api/api` e `/apps/worker/worker` SHALL ser ignorados

### Requirement: Docker Compose para infraestrutura local
O repositório SHALL incluir `docker-compose.yml` na raiz que sobe PostgreSQL, Redis e um storage compatível com S3 (RustFS) prontos para desenvolvimento, com o bucket `ultramedia` criado, CORS liberado para o dashboard local e leitura pública apenas para a saída HLS e as miniaturas.

#### Scenario: Infraestrutura sobe em um comando
- **WHEN** o desenvolvedor executa `make up`
- **THEN** PostgreSQL SHALL estar acessível em `localhost:5432`, Redis em `localhost:6379` e o storage S3 em `localhost:9000`

#### Scenario: Bucket e CORS prontos após make up
- **WHEN** `make up` termina
- **THEN** o bucket `ultramedia` SHALL existir e uma requisição com `Origin: http://localhost:5173` SHALL receber o header `Access-Control-Allow-Origin` correspondente

#### Scenario: Política de leitura pública do HLS
- **WHEN** `make up` termina
- **THEN** objetos em `assets/*/hls/*` e `assets/*/thumbnails/*` SHALL poder ser lidos sem credenciais e os demais SHALL continuar privados

#### Scenario: make up é idempotente
- **WHEN** `make up` é executado de novo com a infraestrutura já de pé
- **THEN** o comando SHALL terminar sem erro e sem apagar o conteúdo do bucket

#### Scenario: Infraestrutura desce em um comando
- **WHEN** o desenvolvedor executa `make down`
- **THEN** todos os containers SHALL ser parados e removidos

### Requirement: Makefile como orquestrador
O repositório SHALL incluir `Makefile` na raiz com targets documentados para as operações essenciais de desenvolvimento, carregando `.env` quando o arquivo existir.

#### Scenario: Targets obrigatórios presentes
- **WHEN** o desenvolvedor executa `make help` ou `make`
- **THEN** os targets `up`, `down`, `migrate`, `migrate-down`, `dev-api`, `dev-worker`, `dev-dashboard`, `build` e `test` SHALL estar listados

#### Scenario: Migrate executa migrações
- **WHEN** o desenvolvedor executa `make migrate` com infraestrutura rodando
- **THEN** todas as migrações pendentes SHALL ser aplicadas ao banco de dados sem exigir a CLI `migrate` instalada globalmente

#### Scenario: Variável ausente gera mensagem clara
- **WHEN** um target que precisa do banco é executado sem `DATABASE_URL` definida
- **THEN** o Makefile SHALL avisar que a variável não está definida e sugerir `cp .env.example .env`

### Requirement: README com setup local
O repositório SHALL incluir `README.md` na raiz explicando pré-requisitos, como subir a infraestrutura, aplicar migrações e rodar API, worker e dashboard.

#### Scenario: Novo desenvolvedor segue o README
- **WHEN** um desenvolvedor segue os passos do README numa máquina com Go, Node.js e Docker
- **THEN** ele SHALL conseguir executar `make up`, `make migrate`, `make dev-api` e `make dev-worker` sem passos extras
