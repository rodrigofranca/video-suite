# Spec Delta

## ADDED Requirements

### Requirement: Progresso da conversão
Durante a conversão, o job SHALL reportar o progresso como a fração da duração já escrita pelo FFmpeg, de 0 a 99, chegando a 100 só quando o job termina. A duração SHALL vir do próprio arquivo baixado, sem depender do job de metadata.

#### Scenario: Progresso crescente
- **WHEN** um vídeo de 8 segundos é convertido
- **THEN** os valores reportados SHALL crescer, ficar entre 0 e 99 e o último SHALL ser pelo menos 80
