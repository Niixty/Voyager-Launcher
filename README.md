# Voyager Launcher — loading nativo para Sunshine / Moonlight

Este projeto mostra uma tela de carregamento no Windows enquanto o jogo iniciado via Sunshine abre. O Moonlight transmite essa tela.

## Estrutura da pasta

- `*.go`, `cmd/`: codigo-fonte do aplicativo e atualizador.
- `tools/`: scripts de instalacao e sincronizacao.
- `.github/workflows/`: compila e publica versoes Windows x64 quando a `main` recebe alteracoes.
- **`app/VoyagerLauncher.exe`** e `app/VoyagerUpdater.exe`: executaveis instalados no proprio clone.
- **`app/Games/`**: configuracoes, capas e fundos de cada jogo.

A pasta **`app/`** esta no `.gitignore` e fica somente no seu PC. Os executaveis e os dados de jogos nao sao enviados para o GitHub Desktop. O atualizador busca executaveis nas Releases do GitHub.

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
