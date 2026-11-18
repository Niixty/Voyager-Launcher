# Voyager Launcher — Loading para Sunshine / Moonlight

Projeto independente da biblioteca RecompLauncher. O Voyager e uma tela de carregamento nativa do Windows para jogos iniciados pelo Sunshine e vistos pelo Moonlight.

## Repositorio

**GitHub:** https://github.com/Niixty/Voyager-Launcher

O `main` e a versao principal. Qualquer commit novo nessa branch dispara o GitHub Actions, que compila Windows x64, executa verificacoes e publica um **Release v0.8.N** automaticamente. Em PCs que ja possuem o Voyager instalado, a atualizacao e baixada e verificada por SHA-256. O programa se atualiza com o helper nativo apos fechar, sem alterar `Games/`.

## Componentes

- `VoyagerLauncher.exe` — gerenciador de jogos + tela de carregamento fullscreen.
- `VoyagerUpdater.exe` — substitui o programa quando ele nao esta em execucao.
- `Games/<id>/config.json`, `grid.png`, `hero.png` — **dados locais e pessoais, nunca enviados para o repositorio**.
- `tools/Sincronizar-Git.ps1` — recebe alteracoes de `origin/main` com `git merge --ff-only`, sem descartar alteracoes locais.
- `tools/Ativar-Sincronizacao.ps1` — registra tarefas no Agendador do Windows.
- `tools/Instalar-Voyager.ps1` — instalacao inicial usando o Release do GitHub (sem precisar de Go no PC).

## Primeira configuracao — via GitHub Desktop

1. No GitHub Desktop, publique um repositorio proprio chamado **Voyager-Launcher** em `Niixty` (nao um fork). Para instalar este codigo, copie o conteudo da pasta do projeto para dentro da pasta do repositorio, mantendo `.github/workflows/windows-release.yml` e `cmd/updater/main.go`.
2. No GitHub Desktop, faca **Commit to main** e **Push origin**. Acesse `https://github.com/Niixty/Voyager-Launcher/actions` e aguarde uma compilacao concluida com sucesso. Isso deve criar automaticamente um Release com os executaveis.
3. Na copia local do repositorio, abra `INSTALAR-VOYAGER.cmd`. Escolha a **pasta onde ja esta seu Voyager** para preservar o comando do Sunshine e os jogos; ou aceite a pasta padrao `%LOCALAPPDATA%\VoyagerLauncher`.
4. O instalador baixa os dois `.exe` da release, valida a integridade, preserva `Games`, e cadastra **duas tarefas**: `Voyager-Git-Sync` (10 minutos) e `Voyager-EXE-Update` (30 minutos).
5. Sunshine continua usando `"C:\SuaPasta\VoyagerLauncher.exe" --game "IDDoJogo"`.

## Como as atualizacoes se propagam

**Codigo-fonte:** edicoes, novos arquivos e exclusoes feitos no GitHub chegam a pasta local clonada pelo GitHub Desktop a cada 10 minutos, **somente** quando voce estiver em `main`, a copia local estiver limpa e o historico aceitar fast-forward. Se houver mudancas locais, o script para e registra a situacao em `%LOCALAPPDATA%\VoyagerLauncher\sync-git.log`. NUNCA usa `reset --hard`, `push` nem `force`.

**Alteracoes feitas no proprio PC:** continuam precisando de **Commit** e **Push origin** no GitHub Desktop. Nenhum arquivo pessoal sera enviado automaticamente.

**Executavel:** cada commit no `main` produz um novo Release. O PC checa a cada 30 minutos de forma silenciosa ou quando o gerenciador e aberto. Quando disponivel, o programa e baixado, conferido e substituido depois que a instancia atual termina. **Durante `--game` nao ha requisicoes de rede nem atraso por atualizacao.** Se nao tiver internet, a versao instalada continua operando.

**Atualizador:** o helper `VoyagerUpdater.exe` nao se atualiza a si mesmo. Se o helper mudar, execute `INSTALAR-VOYAGER.cmd` para atualizar os dois binarios manualmente.

## Seguranca e limites

- Tarefas agendadas fazem apenas atualizacao remota -> local. O GitHub Desktop nao faz push automatico.
- Nenhuma tarefa modifica `Games/`, capas, ROMs ou saves.
- O Git nao consegue aplicar alteracoes remotas se voce tiver arquivos locais alterados sem commit ou branches divergentes; resolva no GitHub Desktop.
- O usuario precisa de acesso de escrita na pasta do programa. Evite `C:\Program Files`.
- O `.exe` nao tem assinatura de codigo. O SHA-256 valida os bytes recebidos, mas nao e substituto de assinatura digital.
- O fluxo foi compilado fora do Windows e precisa de teste real do Agendador de Tarefas e atualizacao no seu PC.

## Desenvolvimento

Go 1.23+:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-H windowsgui' -o VoyagerLauncher.exe .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-H windowsgui' -o VoyagerUpdater.exe ./cmd/updater
```
