# Spec Delta

## ADDED Requirements

### Requirement: Uso de IA no dashboard
Os detalhes do vídeo SHALL mostrar **AI usage**: tokens, custo e a divisão por ação, ou "None yet". A página **AI usage**, no menu, SHALL ter os períodos Today, 7, 30 e 90 dias e All time; os totais de custo, tokens de entrada, tokens de saída e chamadas, com aviso quando há chamadas sem preço ou lidas dos logs; um gráfico de tokens por dia, com os valores de cada dia ao passar o mouse ou focar a barra, e a mesma série como tabela; e tabelas por ação, modelo e vídeo, com link para o vídeo.

#### Scenario: Passar o mouse num dia
- **WHEN** o usuário passa o mouse na barra de um dia
- **THEN** o painel SHALL mostrar os tokens de entrada e de saída, as chamadas e o custo daquele dia
