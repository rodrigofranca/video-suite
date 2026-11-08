# Spec Delta

## ADDED Requirements

### Requirement: Vinhetas de abertura e encerramento
Com `intro` ou `outro`, o job SHALL usar a versão da vinheta no formato do corte (em pé quando a altura passa da largura), ou a outra, inteira sobre uma cópia desfocada dela mesma. SHALL recusar chaves fora de `library/vignettes/` e vinhetas de mais de 30 s. Num só comando FFmpeg, SHALL ajustar cada vinheta ao tamanho e à taxa de quadros do corte, com som estéreo ou silêncio, e emendar abertura, corte e encerramento. O artefato SHALL registrar `preset`, `intro`, `outro` e `totalSeconds`.

#### Scenario: Abertura deitada num corte em pé
- **WHEN** um corte 9:16 de 3 s tem só a versão 16:9 da abertura (1 s) e o encerramento 9:16 (2 s)
- **THEN** o arquivo SHALL ter 1080x1920, cerca de 6 s, com som, e o log SHALL dizer que a abertura foi encaixada

#### Scenario: Vinheta longa demais
- **WHEN** o encerramento tem 31 s
- **THEN** o job SHALL falhar dizendo que vinhetas podem ter até 30 s
