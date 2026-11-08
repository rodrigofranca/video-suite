# Spec Delta

## ADDED Requirements

### Requirement: Apagar pelo dashboard
A página do vídeo SHALL ter, no fim, **Delete video**, e cada corte SHALL ter **🗑 Delete**. Os dois SHALL pedir confirmação no próprio lugar, dizendo o que será apagado e que não dá para desfazer, e SHALL ficar desativados enquanto algum job do vídeo roda. Depois de apagar o vídeo, o painel SHALL voltar para a lista.

#### Scenario: Apagar o vídeo
- **WHEN** o usuário toca **Delete video** e confirma com **Yes, delete the video**
- **THEN** o painel SHALL mostrar a lista de vídeos, sem esse vídeo
