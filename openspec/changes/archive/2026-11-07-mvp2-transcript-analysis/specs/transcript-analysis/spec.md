# Spec Delta

## Purpose
Define o job `analyze-transcript`, que pede a um modelo de linguagem um resumo, tags e capítulos a partir da transcrição do vídeo.

## ADDED Requirements

### Requirement: Resumo, tags e capítulos
O job SHALL ler a transcrição do asset no banco, enviá-la ao gateway (`LLM_BASE_URL`, modelo `LLM_MODEL`) em trechos de cerca de 20 segundos marcados com `[MM:SS]`, e SHALL gravar em `analyses` o resumo, as tags, os capítulos, o idioma da transcrição e o modelo que respondeu. O texto SHALL ser escrito no idioma da transcrição. Rodar de novo SHALL substituir a análise anterior.

#### Scenario: Transcrição em português
- **WHEN** a transcrição tem idioma `pt` e o modelo responde com resumo, tags e dois capítulos
- **THEN** a análise SHALL ser gravada com idioma `pt`, o resumo, as tags e os capítulos

#### Scenario: Modelo atrás de um apelido
- **WHEN** o gateway responde com o cabeçalho `x-litellm-model-name: openai/deepseek-v4-flash`
- **THEN** o modelo gravado SHALL ser `deepseek-v4-flash`

### Requirement: Resposta conferida
O job SHALL aceitar o JSON da resposta mesmo com cercas de código, blocos de raciocínio ou texto em volta. Se a resposta não tiver um `summary` não vazio ou não for JSON, o job SHALL pedir uma correção ao modelo uma única vez e, se a segunda resposta também falhar, SHALL terminar `failed` sem gravar análise.

#### Scenario: Primeira resposta em texto livre
- **WHEN** o modelo responde com uma frase e, no pedido de correção, com JSON válido
- **THEN** a análise SHALL ser gravada a partir da segunda resposta

#### Scenario: Duas respostas inválidas
- **WHEN** as duas respostas não têm `summary`
- **THEN** o job SHALL ficar `failed` com a mensagem `the model's answer could not be used: ...`

### Requirement: Tags e capítulos limpos
As tags SHALL ser no máximo 10, sem `#`, sem repetição (ignorando maiúsculas) e com até 40 caracteres. Cada capítulo SHALL começar no início de trecho mais próximo do horário dado, dentro da duração da fala; os capítulos SHALL ficar em ordem, o primeiro em 0, com pelo menos 10 segundos entre eles e títulos de até 80 caracteres. Menos de dois capítulos SHALL virar uma lista vazia.

#### Scenario: Capítulos bagunçados
- **WHEN** o modelo devolve capítulos fora de ordem, um a 4 segundos do anterior e outro depois do fim
- **THEN** os capítulos gravados SHALL estar em ordem, sem o muito próximo e sem o que passa do fim

#### Scenario: Um capítulo só
- **WHEN** o modelo devolve um único capítulo
- **THEN** a análise SHALL ter `chapters` vazio

### Requirement: Nada a analisar
Sem transcrição (vídeo sem áudio ou transcrição com falha) ou sem fala, o job SHALL terminar `done` sem chamar o modelo e sem gravar análise, registrando o motivo no log.

#### Scenario: Vídeo sem fala
- **WHEN** a transcrição não tem trechos com texto
- **THEN** o job SHALL terminar com o log `nothing to analyze: no speech was found`

### Requirement: Limites e erros do gateway
O job SHALL falhar com mensagem clara quando `LLM_BASE_URL` não estiver definida, quando o prompt passar de `LLM_MAX_INPUT_CHARS` (padrão 200000) ou quando o gateway responder com erro depois das novas tentativas. As mensagens SHALL nunca conter chaves de API.

#### Scenario: Gateway sem chave do provedor
- **WHEN** o gateway responde `AuthenticationError`
- **THEN** o job SHALL falhar na primeira tentativa, com a dica `check the provider key set in the gateway`

#### Scenario: Transcrição longa demais
- **WHEN** o prompt passa do limite
- **THEN** o job SHALL falhar citando `LLM_MAX_INPUT_CHARS`, sem chamar o modelo
