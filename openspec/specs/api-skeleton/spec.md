## ADDED Requirements

### Requirement: API conecta ao PostgreSQL na inicialização
O binário `apps/api` SHALL conectar ao PostgreSQL via pgx na inicialização e falhar rápido se a conexão não for estabelecida.

#### Scenario: API inicia com banco disponível
- **WHEN** `make dev-api` é executado com PostgreSQL rodando e migrações aplicadas
- **THEN** a API SHALL iniciar e logar porta de escuta (padrão: 8080)

#### Scenario: API falha rápido sem banco
- **WHEN** PostgreSQL não está disponível
- **THEN** a API SHALL logar o erro e encerrar com código de saída não-zero

### Requirement: Endpoints stub do MVP1 presentes e roteados
A API SHALL ter os endpoints definidos no PRD roteados e respondendo, mesmo que com corpo stub.

#### Scenario: POST /assets/upload retorna 501
- **WHEN** requisição POST para `/assets/upload` é feita
- **THEN** a API SHALL responder HTTP 501 com body `{"error": "not implemented"}`

#### Scenario: GET /assets retorna lista vazia
- **WHEN** requisição GET para `/assets` é feita com banco vazio
- **THEN** a API SHALL responder HTTP 200 com body `{"assets": []}`

#### Scenario: GET /assets/:id retorna 404 para ID inexistente
- **WHEN** requisição GET para `/assets/00000000-0000-0000-0000-000000000000` é feita
- **THEN** a API SHALL responder HTTP 404 com body `{"error": "not found"}`

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
