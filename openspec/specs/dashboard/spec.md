# dashboard Specification

## Purpose
Define o dashboard web (SvelteKit) onde o usuário envia vídeos, acompanha os jobs e assiste ao resultado em HLS.

## Requirements

### Requirement: Lista de vídeos
A rota `/assets` SHALL listar os assets com miniatura, duração, nome, resolução, status e data, e SHALL atualizar a cada 3 segundos. A raiz `/` SHALL redirecionar para `/assets`.

#### Scenario: Nenhum vídeo
- **WHEN** não há assets
- **THEN** a página SHALL mostrar um convite para enviar um MP4

#### Scenario: API fora do ar
- **WHEN** a API não responde
- **THEN** a página SHALL dizer que não conseguiu falar com a API e em qual endereço

#### Scenario: Vídeo processado
- **WHEN** um asset tem miniatura e metadados
- **THEN** o item SHALL mostrar a miniatura com a duração por cima e a resolução

### Requirement: Upload com progresso
A rota `/upload` SHALL aceitar um arquivo `.mp4` de até 2 GB, mostrar o progresso do envio e abrir a página do vídeo ao terminar. Erros da API SHALL aparecer na tela.

#### Scenario: Arquivo recusado
- **WHEN** a API recusa o arquivo
- **THEN** a mensagem de erro da API SHALL aparecer abaixo do formulário

### Requirement: Página do vídeo
A rota `/assets/{id}` SHALL mostrar status, os jobs com status e duração (ou porcentagem, enquanto rodam), os logs de cada job ao clicar nele, os metadados técnicos e os detalhes do source. Enquanto o asset não estiver `ready` ou `failed`, a página SHALL atualizar a cada 3 segundos.

#### Scenario: Vídeo em processamento
- **WHEN** o asset está `processing`
- **THEN** o lugar do player SHALL mostrar que o vídeo está sendo processado

#### Scenario: Vídeo com falha
- **WHEN** o asset está `failed`
- **THEN** a página SHALL mostrar o erro do job HLS

#### Scenario: Conversão em andamento
- **WHEN** o job HLS está `running` com progresso
- **THEN** o lugar do player SHALL mostrar a miniatura com `Encoding HLS… N%` e o job SHALL ter uma barra de progresso

#### Scenario: Metadados
- **WHEN** os metadados foram extraídos
- **THEN** os detalhes SHALL mostrar duração, resolução, codecs, frame rate e bitrate, com um link para o JSON completo do ffprobe

### Requirement: Player HLS
Quando o asset estiver `ready`, a página SHALL reproduzir a playlist com `hls.js`, ou com o HLS nativo do navegador quando Media Source Extensions não existirem.

#### Scenario: Reprodução
- **WHEN** o usuário abre um vídeo `ready` e aperta play
- **THEN** o vídeo SHALL tocar a partir da playlist pública

### Requirement: Rodar um job de novo pelo dashboard
A página do vídeo SHALL oferecer **Run again** na tentativa mais recente de cada job `done` ou `failed`, visível direto quando o job falhou e ao abrir o job nos demais casos. O botão SHALL ficar oculto enquanto algum job do asset estiver pendente ou rodando. Quando houver mais de uma tentativa de um tipo, cada uma SHALL mostrar seu número, com as antigas esmaecidas.

#### Scenario: Refazer a miniatura
- **WHEN** o usuário abre o job de miniatura e clica em **Run again**
- **THEN** uma nova linha `attempt 2` SHALL aparecer e a página SHALL acompanhar o job até ele terminar
