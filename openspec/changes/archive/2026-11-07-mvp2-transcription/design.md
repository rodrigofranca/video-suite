## Context

Primeira entrega do Milestone 2. Decisões de arquitetura nas ADRs 0016 (worker Python com stream próprio) e 0017 (transcrição depois do HLS, legendas públicas).

## Decisions

**Contratos compartilhados por arquivo:** o Go é a fonte. `go test ./packages/job-contracts -update` grava `testdata/contracts.json`; o teste Go falha se o arquivo estiver desatualizado e o teste Python falha se os valores dele divergirem.

**Ciclo de vida espelhado:** `runner.py` e `queue.py` reproduzem `pipeline.Runner` e `queue.Consumer`: `Retry` (equivalente a `queue.ErrRetry`) deixa a mensagem sem ack; o limite de tempo e o encerramento são verificados entre trechos transcritos, porque a inferência em C++ não pode ser interrompida no meio.

**Áudio separado:** o worker Go grava FLAC mono a 16 kHz lendo o original por URL assinada; o worker de IA baixa só esse arquivo.

**Legendas legíveis:** com `word_timestamps`, os trechos do Whisper (até 30 s) são divididos em legendas de até 6 s e 84 caracteres, em no máximo duas linhas.

**Modelo carregado na inicialização:** o worker falha ao iniciar se o modelo não carregar, em vez de falhar no primeiro job. No Docker, `HF_HOME=/models` fica num volume.

**Compatibilidade:** o faster-whisper 1.2.1 usa um parâmetro que o PyAV 19 removeu, por isso `av<19`. O `uv` resolve só versões publicadas há pelo menos duas semanas (`exclude-newer`).

## Risks / Trade-offs

- **CPU:** o modelo `base` transcreve bem mais rápido que o tempo real em CPU, mas vídeos longos ocupam o worker; modelos maiores são mais precisos e mais lentos.
- **Imagem de ~700 MB** (CTranslate2, onnxruntime, PyAV): aceitável para um worker de ML.
- **Duas implementações do ciclo de vida** (Go e Python) podem divergir; os testes dos dois lados cobrem os mesmos cenários.
- **Um idioma por asset** nesta entrega.
