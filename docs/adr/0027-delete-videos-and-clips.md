# Apagar vídeos e cortes, com todos os arquivos

Até aqui nada podia ser apagado: cada teste deixava o vídeo, os arquivos e os cortes para sempre. Este é o primeiro passo para usar a plataforma de verdade.

A API ganhou `DELETE /assets/{id}` e `DELETE /assets/{id}/clips/{clipId}`, que respondem 204. Apagar um vídeo remove a linha do asset, e o banco apaga em cascata o original, os jobs e seus logs, os artefatos, os metadados, a transcrição, a análise e os cortes. Em seguida, a API remove todos os arquivos sob `assets/{id}/` no storage: original, HLS, miniatura, áudio, legendas e exportações. Apagar um corte remove a linha dele, os registros dos arquivos exportados e a pasta `assets/{id}/clips/{clipId}/`. Os jobs de exportação continuam no histórico.

Como em rodar de novo e exportar, apagar espera nenhum job do vídeo estar pendente ou rodando (409). Assim, um worker nunca escreve num vídeo que acabou de sumir. Um vídeo ainda em upload também não pode ser apagado. Se o upload parou há mais de uma hora (a API reiniciou no meio, por exemplo), ele conta como abandonado e pode ser apagado.

A ordem é banco primeiro, arquivos depois. Assim que a transação termina, o vídeo some para todos. Se o storage falhar no meio, sobram arquivos órfãos, que só ocupam espaço; a API registra no log a pasta onde estão, em vez de falhar o pedido, porque não há mais registro para tentar de novo. A ordem inversa deixaria um vídeo listado sem os arquivos dele. Para um engano não esvaziar o bucket, `DeletePrefix` só aceita a pasta de um asset ou uma pasta dentro dela (`assets/<id>/...`, terminando em barra).

No painel, a página do vídeo ganhou, no fim, a seção **Delete video**, e cada corte, o botão **🗑 Delete**. Os dois pedem confirmação no próprio lugar, dizendo o que vai ser apagado e que não dá para desfazer. Depois de apagar o vídeo, o painel volta para a lista.

Não há lixeira nem como desfazer: o storage local não guarda versões, e uma lixeira pediria regras de retenção que ainda não temos.
