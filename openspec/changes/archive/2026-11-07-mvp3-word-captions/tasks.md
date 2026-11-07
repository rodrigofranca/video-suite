# Tasks

## 1. Palavras na transcrição

- [x] 1.1 Migração `0012_add_transcript_words`; `Transcript.Words` no Go; `save_transcript(words=...)` no Python
- [x] 1.2 O transcritor grava as palavras em ordem, sem espaços, com tempos coerentes

## 2. Legendas palavra a palavra

- [x] 2.1 `captions.ParseWords`, `CutWords`, `Phrases`, `WriteASSWords`, `WordStyleFor`
- [x] 2.2 `caption_style` na entrada do `export-clip`; fallback para linhas com log; estilo no artefato
- [x] 2.3 Testes, inclusive conferindo pelos pixels que a palavra dita fica amarela

## 3. API e dashboard

- [x] 3.1 `captionStyle` no pedido, `wordTimings` na lista, `captionStyle` por exportação
- [x] 3.2 Escolha **Word by word / Lines / Off** e aviso para transcrições antigas

## 4. Entrega

- [x] 4.1 ADR 0021, README, CONTEXT
- [x] 4.2 Teste de ponta a ponta com o modelo real, inclusive transcrevendo de novo um vídeo antigo pelo painel
