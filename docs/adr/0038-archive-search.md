# Busca no acervo pelo que foi dito

A redação precisa achar, em todos os vídeos, onde alguém falou de um assunto e ir direto ao momento. A transcrição de cada vídeo fica em `transcripts`, com os trechos num JSON. Procurar ali exigiria abrir o JSON de cada vídeo a cada busca, sem saber de acentos nem de plurais.

**Decisão: a busca de texto completo do PostgreSQL, trecho por trecho.**

- **Uma linha por trecho** em `transcript_segments` (migração `0020`): vídeo, posição, início, fim, texto, a configuração do idioma e o `tsvector`, com índice GIN. O trecho é a unidade porque tem tempo: cada resultado leva a um momento do vídeo.
- **Um gatilho em `transcripts`** refaz as linhas do vídeo a cada transcrição nova e as apaga com ela; a migração indexou as que já existiam. Quem grava a transcrição (o worker Python, os testes em Go) não precisa saber do índice, e o índice não fica para trás. As linhas são dados derivados, como um índice; nenhuma regra de negócio mora no gatilho.
- **Cada transcrição com as regras do seu idioma.** Uma configuração por idioma, `search_<código>`, para os 28 idiomas com stemmer no PostgreSQL; os outros usam `search_simple`. Todas tiram os acentos antes do stemmer: "votacao" acha "votação", e "reformas" acha "reforma". `search_config` escolhe pela língua da transcrição.
- **A busca não pergunta o idioma de quem digita.** `search_query` lê o texto em cada idioma que o acervo tem e junta as leituras com "ou". Às vezes uma palavra casa por coincidência em outro idioma; é o preço de não pedir o idioma.
- **A sintaxe das buscas na web** (`websearch_to_tsquery`): as palavras (todas), "uma frase" entre aspas, -palavra para deixar de fora e `or` entre alternativas.
- **Ordem:** os vídeos pelo melhor trecho (`ts_rank_cd`) e, entre iguais, o mais novo primeiro. De cada vídeo vêm os 3 melhores trechos, na ordem em que foram ditos, e quantos casam ao todo. As páginas têm 20 vídeos.
- **Palavras marcadas:** `ts_headline`, com a mesma configuração, marca as palavras que casaram. A API devolve o trecho em partes (`{text, match}`), e o painel não insere HTML.

**API:** `GET /search?q=&offset=` devolve `{query, searched, videos, more}`:

- `searched`: quantos vídeos têm transcrição;
- `videos`: cada um com `asset`, `language`, `matches` e `moments` (`start`, `end`, `parts`);
- `more`: há mais vídeos depois destes.

Responde 400 sem texto, com mais de 200 caracteres ou com um `offset` inválido. É uma leitura, então fica na camada HTTP, como as outras (ADR 0033).

**Painel:** **Search** entrou no menu, e a lista de vídeos ganhou uma caixa de busca. A busca fica no endereço (`/search?q=`), então pode ser compartilhada, e voltar no navegador a mantém. Cada momento abre o vídeo com `?t=` e `&q=`:

- o player começa ali;
- a transcrição rola até o trecho e o marca;
- o link do topo volta aos resultados.

**Limites conhecidos:**

- O stemmer do PostgreSQL para português não junta singular e plural das palavras em -ção. Para "votação" e "votações", busque `votação or votações`.
- A busca é por palavras, não por sentido: "inflação" não acha "alta dos preços". Uma busca por sentido, com embeddings, pode vir depois, sobre a mesma tabela de trechos.
- As traduções não entram na busca.
- Uma palavra muito comum lê todos os trechos que casam antes de ordenar. Serve para milhares de vídeos, não para milhões.

**Para o futuro:** o corte a partir de uma citação, próximo item do backlog, parte do momento que a busca encontra.
