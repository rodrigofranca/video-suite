# Spec Delta

## MODIFIED Requirements

### Requirement: Docker Compose para infraestrutura local
O repositório SHALL incluir `docker-compose.yml` na raiz que sobe PostgreSQL, Redis, um storage compatível com S3 (RustFS) e o gateway de modelos de linguagem (LiteLLM) prontos para desenvolvimento, com o bucket `ultramedia` criado, CORS liberado para o dashboard local e leitura pública apenas para a saída HLS e as miniaturas.

#### Scenario: Infraestrutura sobe em um comando
- **WHEN** o desenvolvedor executa `make up`
- **THEN** PostgreSQL SHALL estar acessível em `localhost:5432`, Redis em `localhost:6379`, o storage S3 em `localhost:9000` e o gateway em `localhost:4000`, este só a partir da própria máquina

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
O repositório SHALL incluir `Makefile` na raiz com targets documentados para as operações essenciais de desenvolvimento, carregando `.env` quando o arquivo existir. `make down` SHALL parar também a pilha de demo.

#### Scenario: Targets obrigatórios presentes
- **WHEN** o desenvolvedor executa `make help` ou `make`
- **THEN** os targets `up`, `down`, `demo`, `demo-logs`, `sample`, `sample-speech`, `llm-check`, `migrate`, `migrate-down`, `dev-api`, `dev-worker`, `dev-ai-worker`, `dev-dashboard`, `build`, `test`, `test-go` e `test-ai` SHALL estar listados

#### Scenario: Migrate executa migrações
- **WHEN** o desenvolvedor executa `make migrate` com infraestrutura rodando
- **THEN** todas as migrações pendentes SHALL ser aplicadas ao banco de dados sem exigir a CLI `migrate` instalada globalmente

#### Scenario: Variável ausente gera mensagem clara
- **WHEN** um target que precisa do banco é executado sem `DATABASE_URL` definida
- **THEN** o Makefile SHALL avisar que a variável não está definida e sugerir `cp .env.example .env`
