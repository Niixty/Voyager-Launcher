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


### VibePollo e SteamGridDB

Na tela de um jogo, use a aba **VibePollo** para copiar individualmente o nome,
o caminho da imagem PNG (Grid), a pasta do Voyager e o comando
`"VoyagerLauncher.exe" --game "ID"`. No painel do VibePollo, em
**Aplicativos → Adicionar aplicativo**, deixe **Command** vazio e adicione
o comando do Voyager em **Detached Commands**. O processo do Voyager termina
quando o loading termina, então executá-lo como comando principal pode
encerrar a transmissão antes do jogo. A mesma invocação dá suporte a EXE,
atalhos .lnk e RetroArch conforme o modo cadastrado no jogo.

Em **Configurações → Integração SteamGridDB**, cole a chave de API pessoal
obtida em https://www.steamgriddb.com/profile/preferences/api. A chave é
criptografada pelo Windows (DPAPI), salva no perfil local e não aparece
no GitHub, nos logs nem nos arquivos de jogos. Em **Imagens**, clique
**Procurar na base de dados** ao lado de Hero ou Grid. A pesquisa usa o
nome já salvo para o jogo (sem nome, exige preenchimento), mostra possíveis
correspondências, exibe uma galeria paginada e permite escolher e aplicar
a imagem dentro do aplicativo. O Cartridge permanece exclusivamente manual.
Há limites de tamanho para downloads e a origem da imagem precisa ser
um domínio HTTPS do SteamGridDB.


### Loading integrado à imagem do Cartucho / Grid

A barra horizontal separada não é mais renderizada. O progresso acontece
**dentro dos pixels visíveis** da imagem: começa discretamente escurecido,
clareia e recebe uma tonalidade suave da cor extraída do Hero do jogo, da
esquerda para a direita. O canal alfa original não é modificado: as áreas
transparentes do PNG, inclusive os buracos internos, nunca recebem a cor.
Sem Cartridge, a capa Grid assume exatamente o mesmo efeito.

A partir de 95%, o cartucho ou Grid desacelera até ficar centralizado e
recebe um halo suave gerado a partir do seu contorno real. Ao concluir 100%,
faz um salto e uma pequena ampliação, permanecendo visível por instantes
antes do fade-out da tela. O desenho é feito com sprites em cache e
recriado apenas quando a porcentagem muda.

No editor de loading, o controle **Progresso: 0%–100%** permite inspecionar
o efeito sem abrir o jogo, incluindo a animação e o brilho final.
**Testar loading (10 s)** percorre o progresso inteiro sem executar nada.
Layouts anteriores continuam funcionando, mas o elemento antigo da barra
separada agora é ignorado e não aparece mais no editor.


### Central de atualizações (Settings)

A área **Atualizações e diagnóstico** exibe versão instalada, última
versão disponível, estado da conexão e botões separados por espaçamento.
A aplicação consulta apenas os metadados da última release ao iniciar e
cada vez que Configurações é aberta. A consulta não baixa nem instala nada.

Quando há uma versão nova, aparece o botão **Aplicar atualização** na cor
de destaque do tema ativo. Após confirmação, o Voyager baixa o novo EXE,
mostra a porcentagem do download, valida o SHA-256 e atualiza também o
auxiliar `VoyagerUpdater.exe`. Somente então fecha, substitui o executável
com backup da versão antiga e **reabre automaticamente**. Se o download
ou a verificação falhar, o programa permanece aberto e exibe o erro;
o botão não aparece quando já está atualizado.

O histórico de instalações e o diagnóstico da biblioteca continuam
disponíveis na mesma seção, mas em uma área expansível separada.
A tarefa automática `--check-updates` só verifica se existe uma release,
sem instalar silenciosamente ao fechar o aplicativo.


### Ícone do Windows e biblioteca de comandos para Big Picture

O arquivo `app/assets/VoyagerLauncher.ico` contém o novo ícone do Voyager
em várias resoluções (16–256 px) e é a fonte única de identidade do executável,
do `VoyagerUpdater.exe`, da janela, da barra de tarefas e da interface WebView2.
O workflow Windows gera os recursos `.syso` de ambos os aplicativos a partir
desse mesmo arquivo, evitando ícones diferentes entre os executáveis.

A pasta `desktop/controller-icons/` inclui os **38 PNGs transparentes** do
pacote *Flat Gamepad Icons* de RobTheFiveNine, na resolução 150 DPI:
18 genéricos (direcional, analógicos, L3/R3, Start/Select), 11 Xbox e
9 PlayStation. Para reutilizá-los em novas funções do Big Picture, importe
`desktop/controller-icons.js` antes da tela que deseja renderizar e utilize
`VoyagerGamepadIcons.icon("xb_a")`,
`VoyagerGamepadIcons.button(0, "playstation")` ou
`VoyagerGamepadIcons.stick("right", "up")`.
O registro inclui mapeamento dos botões padrão da Gamepad API (0–16).

O Big Picture mostra ícones para **Navegar**, **Iniciar jogo** e **Voltar**,
detectando automaticamente os controles PlayStation DualShock/DualSense e
usando Xbox para os demais controles compatíveis. Tudo funciona offline.
Créditos e licença CC BY-SA 4.0: `desktop/controller-icons/LICENSE.md`.


### Estabilidade do loading e saída segura da prévia

A prévia de 10 segundos usa um relógio dedicado para atualizar o progresso,
independente das mensagens `WM_TIMER` que podem atrasar quando o Windows
está renderizando. A tecla **Esc** funciona mesmo se o overlay em tela cheia
não estiver com foco. Um *watchdog* no processo isolado do teste fecha a
janela ao completar 10 segundos e força o encerramento desse processo
caso o renderizador não responda. Isso não mata nem abre o jogo.

O processamento do cartucho/Grid preserva a transparência original do PNG,
agora com borda de preenchimento suavizada por *smoothstep*, sem linha
vertical de corte seco. Um halo seguindo a silhueta começa discreto em
0% e aumenta de intensidade continuamente até 100%. O desenho reaproveita
bitmaps em memória, com menos reconstruções GDI+ por alteração de progresso.

### Sincronização de Git sem janelas de terminal

A tarefa `Voyager-Git-Sync`, criada por `ATIVAR-SINCRONIZACAO.cmd`,
agora usa o host **WScript.exe** com o arquivo
`tools/Sincronizar-Git-Silencioso.vbs` e roda o PowerShell sem criar
um console visível sobre jogos. O Voyager também tenta converter
silenciosamente uma tarefa antiga para esse novo modo quando iniciado
normalmente em uma pasta de instalação que tenha a árvore `tools`.

Para instalações cuja pasta não contém os scripts, atualize o repositório
local e execute **ATIVAR-SINCRONIZACAO.cmd** uma vez para registrar a nova
ação; isso substitui a antiga tarefa sem alterar os arquivos do jogo.
A verificação do EXE `Voyager-EXE-Update` continua em segundo plano.


### Animação do cartucho com alvo de 60 FPS (sem travadas ao atualizar o progresso)

A animação do Cartucho/Grid foi separada em duas etapas:
**preparo dos pixels** e **apresentação Win32**. O processamento do
preenchimento colorido (inclusive a borda suave e a transparência do PNG)
agora acontece em uma goroutine independente. Ela sempre recebe a
porcentagem mais recente e descarta valores intermediários, evitando a
acumulação de quadros atrasados.

A thread da janela mantém quatro versões *subpixel* do mesmo sprite em
bitmaps DIB pré-alocados e usa `AlphaBlend` em cada quadro. A memória
desses bitmaps é reaproveitada; o antigo ciclo de recriar contextos GDI+
e três ou quatro sprites a cada porcentagem foi removido do caminho
de animação. O fundo Hero e os textos estáticos permanecem em cache;
alterações de progresso invalidam apenas o cartucho e o campo numérico.

No Windows, a precisão do relógio é elevada temporariamente enquanto
a animação está ativa e restaurada quando a tela é fechada. A meta é
**60 Hz**, mas o FPS real depende do compositor do Windows, da máquina
e do Moonlight. O teste de loading ainda encerra após 10 segundos ou
com Esc e mantém o watchdog contra travamentos.


### Editor de loading: coordenadas livres e prévia confiável

As posições X e Y podem variar entre **-3200** e **+3200** para todos
os elementos, incluindo Cartridge, Grid de fallback e textos. Isso
permite, por exemplo, `Y = -50`, ou mover elementos inteiramente para fora
dos 1600×900 virtuais. A posição é preservada ao salvar e recarregar
e também pode ser alterada arrastando o elemento. Os valores de largura
e altura mantêm seus limites independentes.

O botão **Testar loading (10 s)** agora inicia uma prévia supervisionada
pelo gerenciador, confirma que a janela fullscreen foi realmente criada e
aguarda a saída (10 segundos ou Esc), mostrando o estado no próprio botão.
O teste impede prévias duplicadas e **usa o layout atual do editor**,
inclusive alterações não salvas, por meio de um arquivo temporário.
Esse arquivo é removido quando a prévia termina; o layout cadastrado do
jogo não é alterado pelo teste. Se a janela não abrir, o processo de teste
é encerrado e o erro fica visível no aplicativo.

### Final do cartucho sem mudança brusca de tamanho

A partir de 95%, a posição e o tamanho do Cartucho/Grid são interpolados
a partir do último quadro do movimento vertical, com *smoothstep* e sem
mudanças instantâneas. Ao completar 100%, a confirmação usa um pequeno
salto/ampliação de pulso (`sin²`) com início e fim de velocidade zero.
Se a detecção da janela do jogo recuar para uma etapa anterior, o estado
da animação final é reiniciado para evitar saltos ao voltar a 95%.

A mesma suavização foi aplicada à prévia animada no editor WebView2: o
elemento visual não é recriado a cada mudança de porcentagem, e as
transformações são interpoladas continuamente com `requestAnimationFrame`.


### Editor de loading: seleção múltipla e centralização exata

**Shift + clique** sobre as caixas de Cartucho, Título, Descrição,
Plataforma, Porcentagem e Etapa seleciona várias camadas. Arrastar
qualquer uma das camadas selecionadas move o conjunto, sem alterar as
distâncias relativas; Shift + clique novamente remove uma camada da
seleção. As teclas de seta também movem o grupo (Shift + seta: 10 px).
Desfazer/Refazer registram o movimento em uma única etapa.

A seção **POSIÇÃO RÁPIDA NA TELA** oferece centralizar somente no
eixo horizontal, somente no vertical ou em ambos. Os nove atalhos de
ancoragem também funcionam com grupos: a caixa delimitadora do conjunto
é posicionada sem distorcer os elementos internos. A centralização usa
a dimensão efetiva da caixa, com precisão de **meio pixel** quando a
largura/altura é ímpar (por exemplo, um Cartucho de 305 px fica em
X = 647,5, centralizado exatamente em 800).

A antiga camada **Mensagem** (ID legado `heading`) foi removida
permanentemente dos layouts e do render; ao abrir um layout antigo
ela é descartada, mantendo todas as demais posições e preferências.

No modo **Testar loading (10 s)**, a etapa não anuncia mais o modo
de simulação. Exibe as mesmas mensagens de inicialização comuns,
alternando entre: **Verificando arquivos do jogo**,
**Assoprando cartucho...**, **Trocando bateria...**,
**Aguardando a janela do jogo...**, **Janela do jogo detectada**,
**Confirmando imagem do jogo...**, **Finalizando carregamento...**
e **Jogo pronto**. A prévia nunca inicia o jogo de verdade.


### Cor dominante da Hero no título e no cartucho/Grid

A imagem `hero.png` de cada jogo define automaticamente uma única cor de
destaque, selecionando a **família de cores aproveitáveis mais frequente**
na arte (ignorando preto, branco, transparência e tons quase cinza). O
sistema ajusta a luminosidade para garantir a leitura sobre o fundo escuro.

O título do jogo passa a usar **essa mesma cor** automaticamente, e o
preenchimento horizontal translúcido e o halo do cartucho (ou da Grid,
caso não haja cartucho) reutilizam o mesmo valor RGB. Não existe cálculo
separado para o título, então os dois elementos ficam harmonizados.
A opção de cor do título foi fixada em "Hero" no editor, e layouts antigos
que armazenavam `"color": "white"` são atualizados durante a
normalização, sem alterar fonte, alinhamento nem posição.

Se não houver Hero legível ou houver apenas tons acinzentados, é
utilizada a cor neutra de fallback.
