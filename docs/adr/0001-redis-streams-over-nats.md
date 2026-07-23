# Redis Streams como fila de jobs (não NATS)

O worker precisa de at-least-once delivery e re-claim de jobs em caso de crash. Tanto Redis Streams (XREADGROUP) quanto NATS JetStream atendem esse requisito. Escolhemos Redis porque já é dependência comum em stacks Go+Postgres, reduz o Docker Compose a uma peça a menos e tem cliente maduro (`go-redis`). NATS JetStream tem vantagens reais em visibilidade de fila e persistência nativa, mas o custo de trocar a fila é baixo enquanto o contrato de Job permanecer estável — a migração pode ocorrer após o MVP sem reescrever lógica de negócio.
