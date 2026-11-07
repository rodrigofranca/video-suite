## Context

Terceira etapa da retomada: os dois jobs que eram stubs e o progresso da conversão. As decisões novas estão nas ADRs 0013 (miniaturas públicas) e 0014 (leitura do original por URL assinada).

## Goals / Non-Goals

**Goals:**
- Metadados e miniatura reais sem baixar o vídeo inteiro.
- Progresso visível durante a conversão HLS.
- Nenhuma URL assinada em logs, erros ou banco.

**Non-Goals:**
- Várias miniaturas ou sprite de pré-visualização, progresso do download do original, ABR.

## Decisions

**Resumo + saída completa num só registro:** `technical_metadata.metadata_json` guarda `{"summary": {...}, "ffprobe": {...}}`. O resumo é calculado uma vez no worker; a listagem lê `metadata_json->'summary'` direto no SQL, sem interpretar a saída do ffprobe a cada requisição. A saída completa fica disponível em `GET /assets/{id}/metadata` para depuração.

**Escolha do segundo da miniatura:** usa `AtSecond` do contrato (3 s); se a duração conhecida for menor ou igual, usa o meio do vídeo. Se o FFmpeg não gerar imagem (ele termina com sucesso sem arquivo quando o ponto passa do fim), tenta o primeiro quadro. Sem metadados, o job ainda funciona.

**Progresso:** o FFmpeg roda com `-progress pipe:1`; o tempo escrito (`out_time_us`) dividido pela duração dá a porcentagem, limitada a 99 até o upload terminar. A duração vem de um ffprobe local no arquivo já baixado, para o progresso funcionar mesmo se o job de metadata falhou. O `Runner` grava no máximo uma vez por segundo e ignora valores repetidos.

**Redação de segredos:** `ffmpeg.Options.Redact` troca a URL por `<source>` nas linhas do FFmpeg e nas mensagens de erro; `ffmpeg.Probe` faz o mesmo e substitui `format.filename` pela chave de storage.

## Risks / Trade-offs

- **Arquivos com `moov` no fim** exigem mais requisições parciais → ainda são rápidas (teste com arquivo assim: 0,09 s).
- **Assets antigos não têm metadados nem miniatura** → aparecem sem imagem; não há reprocessamento retroativo nesta change.
- **Progresso não cobre o download do original** → em arquivos grandes a barra fica em 0% enquanto o arquivo baixa; o log do job mostra "downloading source".
