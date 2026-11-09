## Context

Decisões na ADR 0031. Segue o desenho das vinhetas (ADR 0030): biblioteca privada, escolhas no preset, instantâneo no job.

## Decisions

**Tamanho pelo lado menor:** a largura do logo é uma porcentagem do lado menor do corte, então a marca tem o mesmo tamanho em 9:16 e 16:9 e um preset serve aos dois. A margem é de 4% do lado menor.

**Abaixo da legenda, fora das vinhetas:** a imagem entra depois do enquadramento e antes do `ass`, e só no corte.

**Checagem na API:** a API lê formato e tamanho com `image.DecodeConfig` do Go e recusa no envio o que não é PNG ou JPEG, ou passa de 4096 px.

## Risks / Trade-offs

- **Marca parada:** sem animação nem posição livre, só os quatro cantos.
- **Botões dos apps:** nos cortes em pé, os cantos de baixo podem ficar sob os botões dos apps sociais; o padrão é o canto de cima à direita.
