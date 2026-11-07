# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Encerramento gracioso da API
A API SHALL parar de aceitar conexões ao receber SIGINT ou SIGTERM, concluir as requisições em andamento (até 10 segundos) e fechar o pool de conexões.

#### Scenario: Ctrl+C encerra a API
- **WHEN** a API recebe SIGINT
- **THEN** SHALL logar `shutting down` e encerrar com código zero
