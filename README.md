# Voyager Launcher — loading nativo para Sunshine / Moonlight

Este projeto mostra uma tela de carregamento no Windows enquanto o jogo iniciado via Sunshine abre. O Moonlight transmite essa tela.

## Gerenciador desktop

Abra `VoyagerLauncher.exe` para usar a biblioteca em Wails com WebView2. A interface HTML, CSS e JavaScript vem embutida no executável e funciona offline. As chamadas do frontend usam a ponte local do Wails. O overlay de `--game` e o print de `--preview` continuam no Win32/GDI+. `--classic` abre o gerenciador anterior. Ele também é a alternativa para um Windows sem WebView2.

Na Biblioteca, busque por título, plataforma ou ID. Abra um jogo para editar sua execução, conferir imagens, testar ou copiar o comando do Sunshine. Ao arquivar, a pasta inteira vai para `Games-archive` ao lado de `Games`; mova-a de volta para recuperar o jogo. No editor, use o seletor de camadas para alcançar elementos sobrepostos. Os campos numéricos usam a tela virtual de 1600 × 900. Salve antes de usar o print em 50%.

**Herança de layout.** O layout individual tem prioridade sobre `loading_default.json`. Sem nenhum deles, vale o de fábrica. “Padrão para todos os jogos” guarda os layouts individuais em arquivos `.before_global_…json` antes de passar os jogos para a herança. “Padrão para jogos sem layout próprio” preserva as personalizações.

**Testar a versão do PR.** Faça uma cópia da sua pasta `app` em outro local. Feche o Voyager, extraia os dois executáveis do artefato `Voyager-Windows-x64` nessa cópia e abra o launcher. A versão preview mantém a atualização automática pausada. Para testar via Sunshine, crie um aplicativo temporário com o comando mostrado nessa cópia. Para voltar, abra a instalação anterior. O instalador normal continua buscando a release estável.

**Desenvolvimento.** Use Go 1.23, `go test -vet=off ./...` no Windows e `go build -tags production,native_webview2loader -ldflags "-H windowsgui" -o dist/VoyagerLauncher.exe .`. Compile o helper com `go build -ldflags "-H windowsgui" -o dist/VoyagerUpdater.exe ./cmd/updater`. Os testes de interface no workflow usam uma instalação temporária e dados fictícios.

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

A aba AJUSTES e o botao CONFIGURACOES na barra lateral permitem escolher tema **Cinza, Roxo, Verde, Vermelho, Azul, Laranja, Rosa ou Ciano**. A cor escolhida aparece **somente nos destaques do gerenciador**. A barra de progresso e o indicador do loading usam automaticamente uma cor extraida da imagem `hero.png` de cada jogo, independentemente do tema escolhido.

Tambem e possivel mostrar/ocultar porcentagem, descricao, etapa atual, ligar/desligar o desfoque de fundo e escolher escurecimento leve, normal ou forte. Os botoes RESTAURAR, ATUALIZACOES, PRINT 50% e ABRIR APP ficam no rodape dessa aba.

As escolhas sao salvas automaticamente em `app/settings.json` (ignorado pelo Git, preservado nas atualizacoes). `app/settings.example.json` documenta os valores iniciais. Mudar o codigo do menu gera novo EXE via Actions; mudar preferencias no menu nao exige recompilacao.

### Cores da tela de loading por jogo

- O tema do aplicativo (cinza, roxo, verde, vermelho etc.) modifica **apenas o gerenciador**.
- Na tela de loading, a barra de progresso e o ponto de destaque usam a cor predominante mais viva da arte `hero.png` do jogo. Os pixels escuros, brancos ou pouco saturados sao desconsiderados para produzir uma cor visivel.
- A cor e calculada uma vez por inicializacao, sem analisar a imagem em cada quadro.
- O recurso `PRINT 50%` usa exatamente a mesma escolha de cor do loading real.
- Se a imagem nao existir, estiver ilegivel ou for inteiramente cinza, usa um cinza neutro. O estado de erro continua vermelho.
- O desfoque e o escurecimento do fundo continuam configuraveis separadamente, sem mudar a cor extraida do hero.

## Cartridge, seletor de imagens e scroll

A aba IMAGENS tem agora Hero 1920x620, Grid 600x900 e Cartridge 1536x1024. O Cartridge e salvo em `app/Games/<ID>/cartridge.png`, redimensionado sem recorte e com transparencia, mas ainda **nao aparece no loading**.

O dialogo de arquivos usa agora filtros Windows UTF-16 com separadores NUL preservados, evitando fechamento inesperado. O desenho do gerenciador passou para buffer duplo e o scroll so atua sobre a lista lateral de jogos para evitar piscadas.

## Cartridge animado no loading

O loading nao mostra mais os textos `VOYAGER` (topo) e `VOYAGER LAUNCHER` (rodape).
O espaco antes usado pela capa Grid, na lateral esquerda, agora usa a imagem
`app/Games/<ID>/cartridge.png`, preservando as proporcoes e transparencia.
A imagem flutua suavemente para cima e para baixo (aprox. 8 px, ciclo de 2,6 s).
A animacao para quando o loading termina; o comportamento de deteccao do jogo
nao foi alterado. Se nao houver Cartridge, a Grid continua como fallback.
O botao PRINT 50% gera uma imagem estatica usando o Cartridge selecionado,
inclusive antes de salvar. A cor da barra continua sendo extraida do Hero.

## Editor visual de loading por jogo

A aba **EDITOR** mostra uma tela de loading em miniatura e permite mover e dimensionar seus elementos sem editar codigo.
Clique num elemento para selecionar, arraste para reposicionar e arraste os cantos para redimensionar. A roda do mouse ajusta o tamanho do elemento selecionado. Atalhos: setas (1 px), Shift+setas (10 px), Delete (mostrar/ocultar), Ctrl+Z/Ctrl+Y (desfazer/refazer).
Ha grade, snap de 10 px, alinhamento a esquerda/centro/direita/topo/meio/base, ajuste da fonte, opcoes de cores e camadas, restaurar um elemento ou o layout todo, e print de 50%.

**No editor clássico, o layout é individual por jogo**, salvo automaticamente em `app/Games/<ID>/loading_layout.json`, sem modificar `config.json` e sem publicar dados pessoais no Git. Esse JSON e carregado tanto pelo loading real quanto pelo PRINT 50%. Jogos sem layout individual herdam o padrão global ou, na ausência dele, as posições de fábrica. Os elementos editaveis sao Cartridge, plataforma, titulo, descricao, mensagem de loading, porcentagem, barra de progresso e etapa. O tema da interface nao interfere na cor da barra: ela continua usando a cor extraida do Hero.

## Layout padrao compartilhado e correção de flicker

No editor, o botão **PADRÃO P/ TODOS** grava o layout atual em `app/loading_default.json` e aplica imediatamente a todos os jogos existentes. A configuração é herdada automaticamente por jogos novos. As configurações individuais continuam podendo ser editadas depois.
Antes de substituir cada `Games/<ID>/loading_layout.json`, o app guarda uma cópia `loading_layout.json.before_global_<data>.json` na pasta do jogo. O padrão global e os backups são arquivos locais ignorados pelo Git.

O loading real agora utiliza uma superfície de memória reutilizável por monitor e apresenta cada quadro completo de uma vez com BitBlt, sem exibir etapas parciais da pintura GDI+. Isso evita que o fundo e os textos pisquem durante a animação do cartucho. A detecção da janela do jogo, a barra de progresso e a transição final mantêm o comportamento anterior.

## Tempo depois da janela abrir e animação otimizada

Em **AJUSTES → Tempo após abrir o jogo** você pode usar os atalhos de **1 s**, **3 s**, **5 s**, ou ajustar com **-1 / +1 segundo** até o máximo de 15 segundos. O padrão e 1 segundo. O tempo só começa a ser contado depois da detecção contínua da janela do jogo. O restante da detecção, o início do jogo e o fade continuam independentes.

A animação do Cartridge agora solicita quadros a cada 16 ms (aproximadamente 60 FPS, sujeito ao agendamento do Windows) e usa uma oscilação senoidal suave de 11 px em 2,9 segundos. O conteúdo estático de cada monitor — Hero, textos, barras e status — é guardado em uma superfície gráfica reutilizável, recalculado apenas quando os dados mudam. Durante a animação, o programa repõe essa superfície e desenha somente o Cartridge, reduzindo o custo e evitando travamentos associados a redesenhar toda a tela.
O reposicionamento forçado da janela sempre-no-topo também foi reduzido para no máximo uma vez a cada 250 ms.

### Pontos encontrados durante a modernização

| Prioridade | Comportamento anterior | Tratamento na branch |
| --- | --- | --- |
| Alta | A tarefa agendada e o gerenciador compartilhavam os mesmos arquivos temporários. O helper movia o EXE atual antes de conferir a existência do próximo. | Bloqueio por instalação, arquivo temporário exclusivo, SHA-256 conferido novamente no helper, backup e substituição no mesmo volume. |
| Alta | O salvamento podia sobrescrever JSON ilegível e trocar referências de arte personalizadas. | O editor preserva campos adicionais e caminhos de arte. A gravação usa temporário e backup. O cadastro recusa um ID existente. |
| Alta | Uma janela nova sem relação com o jogo podia encerrar o loading. | Execução por atalho solicita o processo ao shell. A detecção considera processo, descendência, caminho ou nome esperado. Launchers que reutilizam outro processo podem exigir “Processo esperado”. |
| Média | A composição em cache desenhava o cartucho sobre elementos que o editor colocava à frente. | O renderer conserva a ordem do documento quando elementos se sobrepõem. O editor usa camadas nativas em cache para animar o cartucho. |
| Média | Controles em coordenadas fixas dificultavam o uso do gerenciador em janelas pequenas. | Biblioteca, formulários e editor em HTML responsivo. O renderer de tela cheia mantém as coordenadas virtuais dos layouts existentes. |

O log da instalação indicava que o arquivo preparado estava ausente. Ele não identifica qual processo o removeu. Os testes de recuperação reproduzem essa condição e conferem o conteúdo do executável e do backup.

### Teste visual de loading (10 segundos)

No editor da tela de loading, clique em **Testar loading (10 s)** ao lado de **Print do layout salvo (50%)**. O aplicativo abre a mesma tela cheia nativa, anima o cartucho e fecha automaticamente após dez segundos, sem iniciar o jogo nem procurar seus processos. A prévia usa o layout salvo do jogo; salve as mudanças do editor antes de testar. Também pode ser acionada com `VoyagerLauncher.exe --test-loading ID_DO_JOGO`.

### Alinhamento e nitidez no editor de loading

A prévia do editor agora é gerada em **1920 × 1080**, inclusive as camadas usadas
na animação do cartucho. Selecione um elemento de texto no editor e configure
**Alinhamento horizontal do texto** (esquerda, centro ou direita) e
**Alinhamento vertical do texto** (topo, centro ou embaixo); esses ajustes
controlam o texto dentro da caixa, sem mover o quadro. A grade **Posição rápida
na tela** tem nove atalhos para mover a caixa inteira aos cantos, centros das
bordas ou centro da tela. O movimento manual continua disponível. Layouts
antigos mantêm o alinhamento original ao topo e à esquerda até serem alterados.
