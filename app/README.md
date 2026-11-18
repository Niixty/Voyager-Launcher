# Voyager — arquivos do aplicativo

Os recursos em `app/themes` e `app/assets` sao visiveis e editaveis diretamente no GitHub.
O programa usa `themes/default.json` para cores e textos do loading ao abrir.

Os executaveis `VoyagerLauncher.exe` e `VoyagerUpdater.exe` sao compilados em GitHub Actions, distribuidos nas Releases e instalados localmente nesta pasta; nao sao rastreados no Git.
`Games/`, ROMs, capas e configuracoes pessoais tambem permanecem locais.

Codigo Go: `overlay.go` (loading), `manager.go` (gerenciador), `update.go` (atualizacao), `internal/appearance` (tema).
