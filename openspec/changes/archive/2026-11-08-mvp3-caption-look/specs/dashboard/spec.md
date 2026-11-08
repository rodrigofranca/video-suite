# Spec Delta

## ADDED Requirements

### Requirement: Aparência das legendas no dashboard
Com legenda escolhida, cada corte SHALL mostrar **Position** (Bottom, Middle, Top), bolinhas de cor só para os estilos com destaque, já na cor do estilo, e **CAPS**, marcado de início só em **One word at a time**. A exportação SHALL usar essas escolhas, e a exportação pronta SHALL resumir o que difere do padrão, como `Word box, top, green, caps`.

#### Scenario: Verde em cima
- **WHEN** o usuário escolhe **Top**, a bolinha verde e exporta em **9:16 crop**
- **THEN** a exportação pronta SHALL mostrar `top` e `green` no resumo e o vídeo SHALL ter a legenda verde em cima
