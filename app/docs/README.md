Configure cores em app/themes/default.json usando #AARRGGBB. O programa le o tema ao iniciar. Alterar o JSON nao exige recompilar o executavel.

O editor de loading fica na aba EDITOR. Configuracoes sao salvas por jogo em `app/Games/<ID>/loading_layout.json` (ignorado pelo Git).
O modelo usa canvas virtual 1600 x 900, coordenadas x/y/width/height, fontes, visibilidade e cores `auto`, `white`, `muted`, `hero`. Os arquivos antigos continuam funcionando com o layout padrao.
