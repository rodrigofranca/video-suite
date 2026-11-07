# Spec Delta

## MODIFIED Requirements

### Requirement: Docker Compose para infraestrutura local
O repositório SHALL incluir `docker-compose.yml` na raiz que sobe PostgreSQL, Redis e um storage compatível com S3 (RustFS) prontos para desenvolvimento, com o bucket `ultramedia` criado, CORS liberado para o dashboard local e leitura pública apenas para a saída HLS.

#### Scenario: Infraestrutura sobe em um comando
- **WHEN** o desenvolvedor executa `make up`
- **THEN** PostgreSQL SHALL estar acessível em `localhost:5432`, Redis em `localhost:6379` e o storage S3 em `localhost:9000`

#### Scenario: Bucket e CORS prontos após make up
- **WHEN** `make up` termina
- **THEN** o bucket `ultramedia` SHALL existir e uma requisição com `Origin: http://localhost:5173` SHALL receber o header `Access-Control-Allow-Origin` correspondente

#### Scenario: Política de leitura pública do HLS
- **WHEN** `make up` termina
- **THEN** objetos em `assets/*/hls/*` SHALL poder ser lidos sem credenciais e os demais SHALL continuar privados

#### Scenario: make up é idempotente
- **WHEN** `make up` é executado de novo com a infraestrutura já de pé
- **THEN** o comando SHALL terminar sem erro e sem apagar o conteúdo do bucket

#### Scenario: Infraestrutura desce em um comando
- **WHEN** o desenvolvedor executa `make down`
- **THEN** todos os containers SHALL ser parados e removidos
