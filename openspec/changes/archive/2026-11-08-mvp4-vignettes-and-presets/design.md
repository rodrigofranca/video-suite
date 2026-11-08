## Context

Decisões na ADR 0030. Escolhas do pedido: a vinheta é um vídeo enviado pronto; cada abertura e encerramento tem uma versão por formato; o preset guarda legenda e vinhetas, e o formato continua no botão de exportar.

## Decisions

**Biblioteca separada dos presets:** a vinheta é enviada uma vez e vários presets a usam; uma vinheta em uso não pode ser apagada. O arquivo fica privado em `library/vignettes/` e é tocado por link assinado.

**Versão por formato, com encaixe:** a exportação usa a versão do formato do corte; sem ela, usa a outra inteira sobre fundo desfocado, o mesmo filtro do `vertical-blur`.

**Uma passada de FFmpeg:** cada vinheta é ajustada ao tamanho e à taxa de quadros do corte, com som estéreo de 48 kHz ou silêncio, e as partes são emendadas com `concat`. O corte não é codificado duas vezes. Início e duração passam a ser opções da entrada de origem.

**Instantâneo no job:** a API copia para o job as chaves das vinhetas e o nome do preset. Mudar ou apagar o preset depois não muda uma exportação já pedida, e o arquivo registra o que usou.

**O pedido vale mais que o preset:** com `presetId`, os campos de legenda que o pedido traz valem mais que os do preset, e os ausentes vêm do preset.

**Limites:** 100 MB conferidos pela API, que não tem FFmpeg, e 30 s conferidos pelo worker, com aviso antecipado no painel.

## Risks / Trade-offs

- **Corte seco:** não há transição entre vinheta e corte; a vinheta traz a sua.
- **Sem login:** presets e biblioteca são compartilhados por todos.
