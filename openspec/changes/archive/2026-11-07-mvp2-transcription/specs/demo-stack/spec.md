# Spec Delta

## MODIFIED Requirements

### Requirement: Um comando para tudo
`make demo` SHALL construir as imagens da API, do worker, do worker de IA e do dashboard e subir infraestrutura, migrações, API, workers e dashboard, terminando só quando todos estiverem prontos. Os modelos Whisper baixados SHALL ficar num volume.

#### Scenario: Primeira execução
- **WHEN** o desenvolvedor executa `make demo` numa máquina só com Docker
- **THEN** o dashboard SHALL responder em `localhost:5173`, a API em `localhost:8080/healthz` e o banco SHALL estar migrado

#### Scenario: Reinício sem novo download
- **WHEN** a pilha é derrubada com `make down` e subida de novo
- **THEN** o worker de IA SHALL carregar o modelo do volume, sem baixá-lo outra vez
