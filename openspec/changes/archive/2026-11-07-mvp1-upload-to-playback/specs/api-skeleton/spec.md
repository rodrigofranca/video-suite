# Spec Delta

## REMOVED Requirements

### Requirement: Endpoints stub do MVP1 presentes e roteados
**Reason**: Upload e rotas de leitura deixaram de ser stubs nesta change.
**Migration**: O comportamento passou para as specs `asset-upload` e `asset-api`. O único stub restante está no requisito "Disparo manual de job ainda não implementado".

## ADDED Requirements

### Requirement: Disparo manual de job ainda não implementado
`POST /assets/{id}/jobs` SHALL continuar roteado como stub até existir o disparo manual de jobs.

#### Scenario: Disparo manual responde 501
- **WHEN** requisição POST para `/assets/{id}/jobs` é feita
- **THEN** a API SHALL responder HTTP 501 com body `{"error": "not implemented"}`

### Requirement: API verifica Redis e storage na inicialização
A API SHALL exigir `REDIS_URL` e as variáveis `S3_*` obrigatórias, e SHALL conferir que o Redis responde e que o bucket existe antes de aceitar requisições.

#### Scenario: Storage fora do ar
- **WHEN** a API inicia sem o storage disponível
- **THEN** SHALL logar o erro com o endereço do storage e encerrar com código não-zero
