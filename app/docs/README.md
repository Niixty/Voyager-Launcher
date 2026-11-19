Configure cores em app/themes/default.json usando #AARRGGBB. O programa le o tema ao iniciar. Alterar o JSON nao exige recompilar o executavel.

O editor de loading fica na aba EDITOR. Configuracoes sao salvas por jogo em `app/Games/<ID>/loading_layout.json` (ignorado pelo Git).
O modelo usa canvas virtual 1600 x 900, coordenadas x/y/width/height, fontes, visibilidade e cores `auto`, `white`, `muted`, `hero`. Os arquivos antigos continuam funcionando com o layout padrao.

Use **PADRÃO P/ TODOS** no editor para salvar o layout compartilhado em `app/loading_default.json`. O aplicativo aplica o desenho aos jogos existentes e usa automaticamente em novos jogos. Layouts antigos são copiados antes da substituição; as configurações por jogo continuam independentes após a aplicação. O JSON global é local e não entra no Git.
