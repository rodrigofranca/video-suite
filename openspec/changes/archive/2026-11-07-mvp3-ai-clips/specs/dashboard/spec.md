# Spec Delta

## ADDED Requirements

### Requirement: Cortes no dashboard
A página do vídeo SHALL mostrar uma seção **Clips** com cada corte sugerido: posição, título, nota, intervalo, duração e motivo, e um botão **Preview** que toca só o intervalo no player. Enquanto a sugestão roda, falhou ou não achou cortes, a seção SHALL dizer isso.

#### Scenario: Prévia
- **WHEN** o usuário clica em **Preview** num corte que começa em 1:08
- **THEN** o player SHALL tocar a partir de 1:08 e pausar no fim do corte

### Requirement: Exportar cortes pelo dashboard
Cada corte SHALL ter um botão por formato (**Original**, **9:16 crop**, **9:16 blur**) e, quando o vídeo tem legendas, a opção **Captions**. Durante a exportação, o formato SHALL mostrar o progresso; ao terminar, o vídeo exportado e o link **Download**. Os botões SHALL ficar desativados enquanto algum job do vídeo estiver pendente ou rodando, e os jobs de exportação SHALL não aparecer na lista de jobs do pipeline.

#### Scenario: Exportar em 9:16 com desfoque
- **WHEN** o usuário clica em **9:16 blur** no primeiro corte
- **THEN** a página SHALL mostrar o progresso e, ao terminar, o vídeo vertical e o link de download
