# api-skeleton Specification

## Purpose
Define o esqueleto da API HTTP em Go: conexão com o PostgreSQL, health check, configuração por variáveis de ambiente e rotas do MVP1 ainda em stub.

## Requirements

### Requirement: API conecta ao PostgreSQL na inicialização
O binário `apps/api` SHALL abrir um pool de conexões (`pgxpool`, seguro para requisições simultâneas) via `packages/media-core/db` na inicialização e falhar rápido se o banco não responder.

#### Scenario: API inicia com banco disponível
- **WHEN** `make dev-api` é executado com PostgreSQL rodando e migrações aplicadas
- **THEN** a API SHALL iniciar e logar a porta de escuta (padrão: 8080, configurável por `API_PORT`)

#### Scenario: API falha rápido sem banco
- **WHEN** PostgreSQL não está disponível
- **THEN** a API SHALL logar o erro e encerrar com código de saída não-zero

#### Scenario: Requisições simultâneas
- **WHEN** várias requisições a `GET /healthz` chegam ao mesmo tempo
- **THEN** todas SHALL ser respondidas com HTTP 200

### Requirement: Health check endpoint
A API SHALL expor `GET /healthz` que retorna 200 quando banco está acessível.

#### Scenario: Health check com banco ok
- **WHEN** GET `/healthz` é chamado com banco disponível
- **THEN** a API SHALL responder HTTP 200 com body `{"status": "ok"}`

### Requirement: Configuração via variáveis de ambiente
A API SHALL ler configuração de variáveis de ambiente sem fallback hardcoded para valores de produção. Valores de dev SHALL estar no `docker-compose.yml` ou `.env.example`.

#### Scenario: Variável obrigatória ausente causa falha na inicialização
- **WHEN** `DATABASE_URL` não está definida
- **THEN** a API SHALL logar erro descritivo e encerrar com código não-zero

### Requirement: Encerramento gracioso da API
A API SHALL parar de aceitar conexões ao receber SIGINT ou SIGTERM, concluir as requisições em andamento (até 10 segundos) e fechar o pool de conexões.

#### Scenario: Ctrl+C encerra a API
- **WHEN** a API recebe SIGINT
- **THEN** SHALL logar `shutting down` e encerrar com código zero

### Requirement: API verifica Redis e storage na inicialização
A API SHALL exigir `REDIS_URL` e as variáveis `S3_*` obrigatórias, e SHALL conferir que o Redis responde e que o bucket existe antes de aceitar requisições.

#### Scenario: Storage fora do ar
- **WHEN** a API inicia sem o storage disponível
- **THEN** SHALL logar o erro com o endereço do storage e encerrar com código não-zero
