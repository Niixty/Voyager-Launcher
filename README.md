# Voyager Launcher — loading nativo para Sunshine / Moonlight

Este projeto mostra uma tela de carregamento no Windows enquanto o jogo iniciado via Sunshine abre. O Moonlight transmite essa tela.

## Estrutura da pasta

- `*.go`, `cmd/`: codigo-fonte do aplicativo e atualizador.
- `tools/`: scripts de instalacao e sincronizacao.
- `.github/workflows/`: compila e publica versoes Windows x64 quando a `main` recebe alteracoes.
- **`app/VoyagerLauncher.exe`** e `app/VoyagerUpdater.exe`: executaveis instalados no proprio clone.
- **`app/Games/`**: configuracoes, capas e fundos de cada jogo.

A pasta **`app/` agora e visivel no GitHub** para temas, recursos e documentacao. Os binarios `.exe` e os dados `app/Games/` continuam ignorados pelo Git, para nao haver conflitos com o atualizador nem expor dados pessoais. Os `.exe` permanecem disponiveis nas **Releases**.

### Edicao por modulos

- `app/themes/default.json`: cores e textos do loading, aplicados quando o Voyager e reiniciado, sem recompilar.
- `internal/appearance/`: modulo Go independente que carrega e valida o tema.
- `overlay.go`, `manager.go`, `update.go`: codigo-fonte por funcao; editar codigo exige nova compilacao automatica nas Actions.

Um executavel compilado nao pode ser separado em pastas editaveis. As fontes e recursos externos sao as partes que editamos; o GitHub gera um novo `.exe` quando o codigo muda.

## Migrar a instalacao antiga para app

Se ja instalou em `%LOCALAPPDATA%\VoyagerLauncher`:
1. Feche o VoyagerLauncher e o VoyagerUpdater.
2. Abra o clone do repositorio no GitHub Desktop e atualize a branch `main` (Fetch/Pull).
3. Execute **`MOVER-APP.cmd`** na raiz do repositorio.
4. Ele copia os executaveis, jogos e configuracoes sem substituir arquivos existentes; mantem a instalacao antiga como backup e atualiza as tarefas agendadas.
5. Abra `app\VoyagerLauncher.exe`. Confira seus jogos.
6. **Se o Sunshine ainda aponta para o EXE antigo, altere para o novo caminho**.

No caminho atual do clone, o comando para um jogo tem este formato:
```text
"D:\Jogos\Emuladores\Voyager Launcher\Voyager-Launcher\app\VoyagerLauncher.exe" --game "IDDoJogo"
```
Ajuste o ID de jogo para aquele cadastrado no gerenciador.

## Instalar do zero

Execute `INSTALAR-VOYAGER.cmd` e pressione Enter para instalar por padrao em `app/`. O instalador baixa a ultima Release, verifica SHA-256, preserva Games e registra as tarefas. O agendador pode ser refeito com `ATIVAR-SINCRONIZACAO.cmd`.

## Como funcionam as atualizacoes

- **Codigo-fonte:** a tarefa `Voyager-Git-Sync` sincroniza a `main` a cada 10 minutos com `fetch` + `merge --ff-only`, apenas quando nao existem alteracoes locais. As alteracoes feitas no PC continuam exigindo Commit + Push pelo GitHub Desktop.
- **Aplicativo:** a tarefa `Voyager-EXE-Update` consulta Releases a cada 30 minutos, e o gerenciador tambem consulta ao abrir. A pasta `app/Games/` permanece intocada.
- **Atualizador auxiliar:** se `VoyagerUpdater.exe` mudar, execute novamente `INSTALAR-VOYAGER.cmd`.

Releases: https://github.com/Niixty/Voyager-Launcher/releases

## Desenvolvimento

Go 1.23+; o GitHub Actions executa testes e compila os dois executaveis no Windows. O `exe` nao possui assinatura digital. O SHA-256 verifica a integridade, mas nao substitui uma assinatura de codigo.

## Icone do aplicativo

O icone oficial fica em `app/assets/VoyagerLauncher.ico`, criado a partir da arte enviada pelo autor. A compilacao Windows gera `rsrc_windows_amd64.syso` com `github.com/akavel/rsrc@v0.10.2`, incorporando o icone ao executavel. A janela principal tambem usa esse recurso. O formato `.ico` e um arquivo binario versionado; os executaveis continuam nas Releases.

## Configurações (nova aba AJUSTES)

A aba AJUSTES e o botao CONFIGURACOES na barra lateral permitem escolher tema **Cinza, Roxo, Verde, Vermelho, Azul, Laranja, Rosa ou Ciano**. A cor escolhida aparece nos destaques do gerenciador e na barra/identificacao do loading.

Tambem e possivel mostrar/ocultar porcentagem, descricao, etapa atual, ligar/desligar o desfoque de fundo e escolher escurecimento leve, normal ou forte. Os botoes RESTAURAR, ATUALIZACOES, PRINT 50% e ABRIR APP ficam no rodape dessa aba.

As escolhas sao salvas automaticamente em `app/settings.json` (ignorado pelo Git, preservado nas atualizacoes). `app/settings.example.json` documenta os valores iniciais. Mudar o codigo do menu gera novo EXE via Actions; mudar preferencias no menu nao exige recompilacao.
