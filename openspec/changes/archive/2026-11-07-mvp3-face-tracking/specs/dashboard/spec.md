# Spec Delta

## MODIFIED Requirements

### Requirement: Exportar cortes pelo dashboard
Cada corte SHALL ter um botão por formato (**Original**, **9:16 crop**, **9:16 face**, **9:16 blur**). Durante a exportação, o formato SHALL mostrar o progresso; ao terminar, o vídeo exportado, o link **Download** e o estilo das legendas. Os botões SHALL ficar desativados enquanto algum job do vídeo estiver pendente ou rodando, e os jobs de exportação SHALL não aparecer na lista de jobs do pipeline.

#### Scenario: Exportar em 9:16 com desfoque
- **WHEN** o usuário clica em **9:16 blur** no primeiro corte
- **THEN** a página SHALL mostrar o progresso e, ao terminar, o vídeo vertical e o link de download

## ADDED Requirements

### Requirement: Rosto no dashboard
Antes de exportar, **9:16 face** SHALL dizer em quanto do corte há rosto (`Face in 85% of the clip`), ou que vai ficar no centro porque o rastreio não rodou ou achou poucos rostos. Uma exportação que ficou no centro SHALL mostrar `centred`. A página SHALL buscar os cortes de novo quando o rastreio termina.

#### Scenario: Rastreio terminado
- **WHEN** o job `track-faces` termina com rosto em 100% do primeiro corte
- **THEN** o primeiro corte SHALL mostrar `Face in 100% of the clip` em **9:16 face**, sem recarregar a página
