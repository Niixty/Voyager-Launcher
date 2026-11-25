(async function () {
  const probe = window.go.main.AcceptanceProbe;
  const wait = async (fn, label) => {
    for (let i = 0; i < 120; i++) {
      if (fn()) return;
      await new Promise((r) => setTimeout(r, 100));
    }
    throw Error(label + " — " + document.body.innerText);
  };
  const click = (label) => {
    const b = [...document.querySelectorAll("button")].find(
      (b) => b.textContent === label,
    );
    if (!b) throw Error("Botão ausente: " + label);
    b.click();
  };
  const heading = (title) =>
    document.querySelector("h1")?.textContent === title;
  try {
    await probe.Progress("begin");
    await wait(() => heading("Biblioteca"), "Abrir biblioteca");
    click("+ Adicionar jogo");
    const inputs = document.querySelectorAll("form input");
    const set = (input, v) => {
      input.value = v;
      input.dispatchEvent(new Event("input", { bubbles: true }));
    };
    set(inputs[0], "Smoke");
    set(inputs[1], "Órbita");
    set(inputs[2], "Windows");
    const mode = document.querySelector("form select");
    mode.value = "executavel";
    mode.dispatchEvent(new Event("change"));
    set(
      document.querySelector(".fields.panel input"),
      "C:\\Windows\\notepad.exe",
    );
    document.querySelector("form").requestSubmit();
    await wait(() => heading("Órbita"), "Salvar cadastro");
    click("Imagens");
    for (const kind of ["Hero", "Grid", "Cartridge"]) {
      const choosing = probe.ChooseFile();
      click("Selecionar " + kind);
      await choosing;
      await wait(
        () => document.querySelector("dialog[open] img")?.naturalWidth > 0,
        "Prévia " + kind,
      );
      click("Salvar imagem");
      await wait(
        () => !document.querySelector("dialog[open]"),
        "Salvar " + kind,
      );
    }
    const searches = [...document.querySelectorAll("button")].filter(
      (b) => b.textContent === "Procurar na base de dados",
    );
    if (searches.length !== 2) throw Error("Hero/Grid precisam oferecer SteamGridDB, mas Cartridge não.");
    click("VibePollo");
    await wait(
      () => document.body.innerText.includes("Detached Commands / Comandos em segundo plano"),
      "Instruções VibePollo",
    );
    if (!document.body.innerText.includes("Image / Imagem (PNG)"))
      throw Error("Caminho da capa no VibePollo ausente");
    await probe.Progress("before-editor");
    click("Editor de loading");
    await wait(
      () =>
        document
          .querySelector(".stage")
          ?.style.backgroundImage.includes("data:image/png"),
      "Composição nativa",
    );
    await probe.Progress("editor-ready");
    // Four independent per-game visuals must survive saving a layout.
    const appearance = [...document.querySelectorAll(".appearanceControl input[data-appearance]")];
    if (appearance.length !== 4) throw Error("Aparência do loading não tem quatro controles");
    for (const [key, value] of [
      ["heroBrightness", 75],
      ["heroBlur", 0],
      ["titleBrightness", 145],
      ["cartridgeBrightness", 175],
    ]) {
      const slider = appearance.find((input) => input.dataset.appearance === key);
      if (!slider) throw Error("Controle visual ausente: " + key);
      set(slider, String(value));
      slider.dispatchEvent(new Event("change", { bubbles: true }));
      const label = slider.closest(".appearanceControl")?.querySelector("strong");
      if (!label || label.textContent !== value + "%") throw Error("Porcentagem visual incorreta: " + key);
    }
    const layer = [...document.querySelectorAll(".panel select")].find((select) =>
      [...select.options].some((option) => option.value === "title"),
    );
    const stage = document.querySelector(".stage");
    // Clicking an empty area clears selection without clearing any layout.
    stage.dispatchEvent(new PointerEvent("pointerdown", {
      bubbles: true, cancelable: true, pointerId: 200,
    }));
    if (stage.querySelectorAll(".node.selected").length !== 0 || layer.value !== "")
      throw Error("Clicar fora das camadas não remove a seleção");
    if (document.querySelector(".panel .fields input"))
      throw Error("Propriedades de uma camada ainda aparecem sem seleção");
    // Every layer must be activatable even when hidden from the canvas.
    const descriptionSwitch = document.querySelector('.layerToggles [data-layer-id="description"] input');
    if (!descriptionSwitch) throw Error("Checkbox de ativar camada Descrição ausente");
    descriptionSwitch.click();
    if (stage.querySelector('.node[data-id="description"]'))
      throw Error("Camada Descrição continuou visível após desativar");
    document.querySelector('.layerToggles [data-layer-id="description"] input').click();
    if (!stage.querySelector('.node[data-id="description"]'))
      throw Error("Camada Descrição não reapareceu ao ativar");
    // The stage editor must cover real-game startup texts AND preview texts.
    const messages = [...document.querySelectorAll('.stageMessageEditor input[data-stage-index]')];
    if (messages.length !== 10) throw Error("Editor de etapas não contém as 10 mensagens");
    const originalText = messages[1].value;
    if (!originalText.includes("cartucho")) throw Error("Etapas originais não carregadas");
    set(messages[1], "Soprando cartucho do meu jeito...");
    messages[1].dispatchEvent(new Event("change", { bubbles: true }));
    const previewButtons = document.querySelectorAll(".stageMessageEditor button");
    const previewStage = [...previewButtons].find((button) => button.title.includes("15%"));
    if (!previewStage) throw Error("Botão de prévia individual da etapa ausente");
    previewStage.click();
    if (!document.querySelector(".previewProgress")?.textContent.includes("15%"))
      throw Error("Prévia da etapa não selecionou a porcentagem correspondente");
    layer.value = "title";
    layer.dispatchEvent(new Event("change", { bubbles: true }));
    if (!document.body.innerText.includes("Cor automática da Hero"))
      throw Error("O editor não indica que o Título usa a cor da Hero");
    const textAlign = document.querySelector('[data-text-align="horizontal"]');
    if (!textAlign) throw Error("Alinhamento horizontal ausente");
    textAlign.value = "right";
    textAlign.dispatchEvent(new Event("change", { bubbles: true }));
    const verticalAlign = document.querySelector('[data-text-align="vertical"]');
    if (!verticalAlign) throw Error("Alinhamento vertical ausente");
    verticalAlign.value = "bottom";
    verticalAlign.dispatchEvent(new Event("change", { bubbles: true }));
    const anchors = document.querySelectorAll(".anchorPresets button");
    if (anchors.length !== 9) throw Error("Atalhos de posição ausentes");
    const props = document.querySelector(".panel .fields input");
    set(props, "-50");
    props.dispatchEvent(new Event("change"));
    click("Desfazer");
    click("Refazer");

    // X/Y outside 1600×900 must remain editable and survive save/load.
    layer.value = "cartridge";
    layer.dispatchEvent(new Event("change", { bubbles: true }));
    const cartridgeFields = document.querySelectorAll(".panel .fields input");
    const cartridgeY = cartridgeFields[1];
    if (!cartridgeY || cartridgeY.min !== "-3200")
      throw Error("Campo Y do Cartucho não aceita valores negativos");
    set(cartridgeY, "-50");
    cartridgeY.dispatchEvent(new Event("change", { bubbles: true }));
    if (typeof api.TestLoading !== "function")
      throw Error("Método TestLoading não registrado na interface Wails");
    if (![...document.querySelectorAll("button")].some(b => b.textContent === "Testar loading (10 s)"))
      throw Error("Botão Testar loading ausente");
    await probe.Progress("before-save");
    click("Salvar layout");
    await wait(
      () => document.querySelector("#notice").textContent === "Layout salvo",
      "Salvar layout",
    );
    await probe.Progress("saved");
    const saved = await api.Layout("Smoke");
    if (saved.heroBrightness !== 75 || saved.heroBlur !== 0 ||
        saved.titleBrightness !== 145 || saved.cartridgeBrightness !== 175) {
      throw Error("Brilho/blur de Hero, título e cartucho não persistiram separadamente");
    }
    if (saved.stageTexts?.length !== 10 || saved.stageTexts[1] !== "Soprando cartucho do meu jeito...") {
      throw Error("Mensagens personalizadas das etapas não foram salvas no layout");
    }
    const titleSaved = saved.elements.find((n) => n.id === "title");
    if (titleSaved.color !== "hero")
      throw Error("O Título não foi salvo usando cor automática da Hero");
    if (titleSaved.textAlign !== "right" || titleSaved.verticalAlign !== "bottom") {
      throw Error("Alinhamento não persistido no layout");
    }
    if (titleSaved.x !== -50 || saved.elements.find(n => n.id === "cartridge").y !== -50)
      throw Error("Coordenadas negativas foram perdidas ao salvar o layout");
    await probe.Progress("before-shift");
    // Shift multi-selection must move all selected layers as a group.
    if ([...layer.options].some((option) => option.value === "heading"))
      throw Error("Camada Mensagem ainda aparece no editor");
    const titleNode = stage.querySelector('.node[data-id="title"]');
    if (!titleNode) throw Error("Camada Título ausente");
    titleNode.dispatchEvent(new PointerEvent("pointerdown", {
      bubbles: true, cancelable: true, pointerId: 99, shiftKey: true,
    }));
    await probe.Progress("shift-dispatched");
    if (stage.querySelectorAll(".node.selected").length !== 2)
      throw Error("Shift + clique não selecionou duas camadas");
    const rect = (id) => {
      const node = stage.querySelector('.node[data-id="' + id + '"]');
      return {
        x: parseFloat(node.style.left) * 16,
        y: parseFloat(node.style.top) * 9,
        w: parseFloat(node.style.width) * 16,
        h: parseFloat(node.style.height) * 9,
      };
    };
    const originalTitle = rect("title");
    const originalCartridge = rect("cartridge");
    const tolerance = 0.005;
    const centered = (axis, span) => {
      const a = rect("title"), b = rect("cartridge");
      const start = Math.min(a[axis], b[axis]);
      const end = Math.max(a[axis] + a[span], b[axis] + b[span]);
      return (start + end) / 2;
    };
    await probe.Progress("before-center-x");
    click("Centralizar só horizontal");
    if (Math.abs(centered("x", "w") - 800) > tolerance)
      throw Error("Centralização horizontal do grupo está incorreta");
    if (Math.abs(rect("title").y - originalTitle.y) > tolerance ||
        Math.abs(rect("cartridge").y - originalCartridge.y) > tolerance)
      throw Error("Centralização horizontal alterou o eixo Y");
    if (Math.abs((rect("cartridge").x - rect("title").x) -
                 (originalCartridge.x - originalTitle.x)) > tolerance)
      throw Error("Seleção múltipla perdeu o espaçamento entre camadas");
    const positionAfterHorizontal = rect("title").x;
    await probe.Progress("before-center-y");
    click("Centralizar só vertical");
    if (Math.abs(centered("y", "h") - 450) > tolerance)
      throw Error("Centralização vertical do grupo está incorreta");
    if (Math.abs(rect("title").x - positionAfterHorizontal) > tolerance)
      throw Error("Centralização vertical alterou o eixo X");
    // A single odd-width cartridge needs a true half-pixel center.
    layer.value = "cartridge";
    layer.dispatchEvent(new Event("change", { bubbles: true }));
    click("Centralizar só horizontal");
    const single = rect("cartridge");
    if (Math.abs(single.x + single.w / 2 - 800) > tolerance)
      throw Error("Camada ímpar não centralizou com precisão");
    // Group alignment deliberately dirties the layout. Save it before
    // leaving the editor; otherwise the navigation confirmation modal
    // blocks this automated driver (not an application rendering freeze).
    await probe.Progress("before-group-save");
    click("Salvar layout");
    await wait(() => document.querySelector("#notice").textContent === "Layout salvo",
      "Salvar alinhamento de grupo");
    await probe.Progress("group-saved");

    await probe.Progress("before-capture");
    await probe.Capture("editor");
    await probe.Progress("after-capture");
    await probe.Progress("before-settings");
    click("Configurações");
    await wait(() => heading("Configurações"), "Configurações");
    const updatePanel = document.querySelector(".update-panel");
    if (!updatePanel) throw Error("Central de atualizações não encontrada");
    const applyUpdate = [...updatePanel.querySelectorAll("button")].find(b => b.textContent === "Aplicar atualização");
    if (!applyUpdate) throw Error("Botão Aplicar atualização ausente");
    const updateInfo = await api.UpdateInfo();
    if (updateInfo.preview && !applyUpdate.hidden)
      throw Error("Botão Aplicar atualização deve ficar oculto na versão de testes");
    const actionGap = parseFloat(getComputedStyle(updatePanel.querySelector(".update-actions")).columnGap);
    if (!Number.isFinite(actionGap) || actionGap < 8)
      throw Error("Os botões de atualização continuam sem espaçamento");
    click("Salvar preferências");
    await wait(
      () =>
        document.querySelector("#notice").textContent === "Preferências salvas",
      "Salvar preferências",
    );
    await probe.Progress("before-library-return");
    click("Biblioteca");
    await wait(
      () => document.querySelector(".card strong")?.textContent === "Órbita",
      "Retornar à biblioteca",
    );
    await probe.Capture("biblioteca");
    await api.CopyGame("Smoke", "Smoke2");
    await library();

    // Simulate SteamGridDB replies arriving OUT OF ORDER. Results for game A
    // must never replace those for game B after a game-picker change.
    const steamMethods = {
      saved: api.SteamGridSavedGameTitle,
      key: api.SteamGridKeyConfigured,
      games: api.SteamGridGames,
      images: api.SteamGridImages,
    };
    const pendingImageRequests = [];
    try {
      api.SteamGridSavedGameTitle = async () => "Órbita";
      api.SteamGridKeyConfigured = async () => true;
      api.SteamGridGames = async () => [
        {id: 101, name: "Jogo A"}, {id: 202, name: "Jogo B"},
      ];
      api.SteamGridImages = (id, kind, pageNumber) =>
        new Promise((resolve) => pendingImageRequests.push({id,kind,pageNumber,resolve}));
      const modalPromise = steamGridBrowser(games[0], "grid");
      await wait(() => pendingImageRequests.length === 1,
        "Busca da primeira seleção SteamGridDB");
      const selector = document.querySelector(".sgdb-dialog .sgdb-game-picker select");
      if (!selector) throw Error("Seletor de jogos SteamGridDB ausente");
      selector.value = "202";
      selector.dispatchEvent(new Event("change", {bubbles:true}));
      await wait(() => pendingImageRequests.length === 2,
        "Nova busca do SteamGridDB não iniciou enquanto anterior estava pendente");
      const artwork = (id) => ({
        items: [{url:"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO/JhX8AAAAASUVORK5CYII=",thumb:"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO/JhX8AAAAASUVORK5CYII=",width:id,height:900}],
        total:1,limit:50,
      });
      pendingImageRequests[1].resolve(artwork(202));
      await wait(() => document.querySelectorAll(".sgdb-gallery .sgdb-item").length === 1,
        "Imagens da segunda seleção");
      pendingImageRequests[0].resolve(artwork(101));
      await modalPromise;
      await new Promise((r) => setTimeout(r, 150));
      if (document.querySelectorAll(".sgdb-gallery .sgdb-item").length !== 1 ||
          !document.querySelector(".sgdb-gallery .sgdb-item img")?.alt.includes("202 × 900"))
        throw Error("Galeria antiga sobrescreveu o jogo selecionado");
      document.querySelector(".sgdb-dialog").close();
    } finally {
      api.SteamGridSavedGameTitle = steamMethods.saved;
      api.SteamGridKeyConfigured = steamMethods.key;
      api.SteamGridGames = steamMethods.games;
      api.SteamGridImages = steamMethods.images;
      document.querySelector(".sgdb-dialog")?.close();
    }
    // Visual header follows the platform of the selected game, not the
    // generic "Big Picture" label. Use two different platform names.
    const sortedGames = [...games].sort((a,b) =>
      (a.Config.title || a.ID).localeCompare(b.Config.title || b.ID, "pt-BR"));
    sortedGames[0].Config.platform = "Super Nintendo";
    sortedGames[1].Config.platform = "Nintendo 64";
    const buttons = Array.from({length: 16}, () => ({pressed:false}));
    const pad = {mapping:"standard", connected:true, buttons, axes:[0, 0]};
    Object.defineProperty(navigator, "getGamepads", {configurable:true, value:() => [pad]});
    Object.defineProperty(document, "hasFocus", {configurable:true, value:() => true});
    const launches = [];
    const originalRun = api.Run;
    api.Run = async (id, mode) => launches.push([id, mode]);
    await probe.Progress("before-bigpicture");
    click("Big Picture");
    await wait(() => document.querySelectorAll(".bp-card").length === 2, "Capas Big Picture");
    if (document.querySelector(".bp-heading h1")?.textContent !== "Super Nintendo")
      throw Error("Cabeçalho do Big Picture não mostra a plataforma selecionada");
    const iconAPI = window.VoyagerGamepadIcons;
    if (!iconAPI || iconAPI.names.length !== 38)
      throw Error("Registro completo de ícones genéricos/Xbox/PlayStation ausente");
    if (iconAPI.buttonName(0, "xbox") !== "xb_a" || iconAPI.buttonName(0, "playstation") !== "ps_x")
      throw Error("Mapeamento dos botões Xbox/PlayStation está incorreto");
    if (iconAPI.buttonName(8, "xbox") !== "xb_select" ||
        iconAPI.buttonName(8, "playstation") !== "generic_select")
      throw Error("Select deve usar os ícones de Xbox/PlayStation correspondentes");
    await wait(() => document.querySelector(".bp-open-settings img[src*='xb_select.png']"),
      "Ícone Select no botão Configurações");
    await wait(() => document.querySelector(".bp-controls img[src*='xb_select.png']"),
      "Ícone Select nas dicas de controle");
    await wait(() => document.querySelector(".bp-controls img[src*='xb_a.png']"), "Botão Xbox A");
    const gamepadIcon = await fetch("/controller-icons/ps_x.png");
    if (!gamepadIcon.ok || !(await gamepadIcon.blob()).size)
      throw Error("Imagem PlayStation não foi embutida no EXE");
    pad.id = "DualSense Wireless Controller";
    await wait(() => document.querySelector(".bp-controls img[src*='ps_x.png']"), "Ícones PlayStation automáticos");
    pad.id = "Xbox Wireless Controller";
    await wait(() => document.querySelector(".bp-controls img[src*='xb_a.png']"), "Retorno aos ícones Xbox");
    buttons[15].pressed = true;
    await wait(() => document.querySelectorAll(".bp-card")[1].matches(".selected"), "Direcional");
    if (document.querySelector(".bp-heading h1")?.textContent !== "Nintendo 64")
      throw Error("Título não atualiza quando muda a plataforma do jogo selecionado");
    if (typeof api.BigPictureAccent !== "function")
      throw Error("Método que extrai cor dominante da Hero não registrado no Wails");
    const heroAccent = (await api.BigPictureAccent("Smoke2")).toUpperCase();
    await wait(() => document.querySelector(".big-picture")?.style.getPropertyValue("--bp-glow-color") === heroAccent,
      "Brilho do cartucho deve usar cor dominante da Hero do jogo selecionado");
    buttons[15].pressed = false;
    buttons[0].pressed = true;
    await wait(() => launches.length > 0, "Botão A");
    await new Promise(r => setTimeout(r, 1150));
    if (launches.length !== 1 || launches[0][0] !== "Smoke2" || launches[0][1] !== "--game")
      throw Error("O botão A deve abrir uma vez o jogo selecionado");
    buttons[0].pressed = false;
    await new Promise((r) => setTimeout(r, 100));

    // The second gamepad must work while the first is connected and idle.
    // It was previously ignored because the polling loop broke at pad 0.
    const secondButtons = Array.from({length: 16}, () => ({pressed:false}));
    const secondPad = {
      id:"DualSense Wireless Controller", mapping:"standard",
      connected:true, buttons:secondButtons, axes:[0,0],
    };
    Object.defineProperty(navigator, "getGamepads", {configurable:true, value:() => [pad, secondPad]});
    // Start must no longer open Settings; Select does, on either pad.
    secondButtons[9].pressed = true;
    await new Promise((r) => setTimeout(r, 180));
    if (!document.querySelector(".bp-settings").hidden)
      throw Error("Start ainda abre configurações; o atalho correto é Select");
    secondButtons[9].pressed = false;
    await new Promise((r) => setTimeout(r, 110));
    secondButtons[8].pressed = true;
    await wait(() => !document.querySelector(".bp-settings").hidden, "Select no segundo controle abre configurações");
    secondButtons[8].pressed = false;
    await new Promise((r) => setTimeout(r, 120));
    if (!document.querySelector(".bp-controls img[src*='ps_x.png']"))
      throw Error("Glifos não acompanharam o segundo controle PlayStation");
    if (!document.querySelector(".bp-controls img[src*='generic_select.png']") ||
        !document.querySelector(".bp-open-settings img[src*='generic_select.png']"))
      throw Error("Ícone Select genérico ausente para controles PlayStation");
    // The controller's South button (A on Xbox / Cross on PS) must
    // ACTIVATE the selected settings row, not launch the underlying game.
    if (document.activeElement?.dataset.setting !== "view")
      throw Error("Foco inicial das configurações deveria estar em Imagens");
    secondButtons[0].pressed = true;
    await wait(() => document.querySelector(".big-picture").classList.contains("bp-cartridges"),
      "Botão A não alterou a opção Imagens dentro das configurações");
    secondButtons[0].pressed = false;
    await new Promise((r) => setTimeout(r, 110));
    if (launches.length !== 1)
      throw Error("Botão A nas configurações abriu um jogo indevidamente");
    // Move to the button layout row and force Xbox then PlayStation.
    secondButtons[13].pressed = true;
    await wait(() => document.activeElement?.dataset.setting === "buttonStyle",
      "Direcional não focou Padrão dos botões");
    secondButtons[13].pressed = false;
    await new Promise((r) => setTimeout(r, 110));
    secondButtons[0].pressed = true;
    await wait(() => document.querySelector(".big-picture")?.dataset.controller === "xbox",
      "Botão A não selecionou ícones Xbox");
    secondButtons[0].pressed = false;
    await new Promise((r) => setTimeout(r, 110));
    if (!document.querySelector(".bp-controls img[src*='xb_a.png']"))
      throw Error("Forçar ícones Xbox não alterou os botões visuais");
    secondButtons[0].pressed = true;
    await wait(() => document.querySelector(".big-picture")?.dataset.controller === "playstation",
      "Botão A não selecionou ícones PlayStation");
    secondButtons[0].pressed = false;
    await new Promise((r) => setTimeout(r, 110));
    if (!document.querySelector(".bp-controls img[src*='ps_x.png']"))
      throw Error("Forçar ícones PlayStation não alterou os botões visuais");
    const toggles = [
      ["background", "solid"], ["showTitle", false], ["showPlatform", false],
    ];
    for (const [key] of toggles) {
      const option = document.querySelector('.bp-setting[data-setting="' + key + '"]');
      if (!option) throw Error("Opção do Big Picture ausente: " + key);
      option.click();
    }
    await wait(() => document.querySelector(".big-picture").classList.contains("bp-cartridges"),
      "Visualização de cartucho");
    // The cartridge image must sit directly over the Hero without a
    // dark rectangular background or filled selected card.
    const cartridgeCard = document.querySelector(".bp-cartridges .bp-card.selected");
    const cartridgeCover = cartridgeCard?.querySelector(".bp-cover");
    const transparent = (node) => ["transparent","rgba(0, 0, 0, 0)"].includes(
      getComputedStyle(node).backgroundColor);
    if (!cartridgeCard || !cartridgeCover ||
        !transparent(cartridgeCard) || !transparent(cartridgeCover)) {
      const cardBg = cartridgeCard ? getComputedStyle(cartridgeCard).backgroundColor : "missing";
      const coverBg = cartridgeCover ? getComputedStyle(cartridgeCover).backgroundColor : "missing";
      const cls = document.querySelector(".big-picture")?.className || "missing";
      throw Error("Cartucho deveria ser transparente: card=" + cardBg +
        " cover=" + coverBg + " mode=" + cls);
    }
    if (getComputedStyle(cartridgeCard).outlineStyle !== "none")
      throw Error("A seleção do cartucho ainda desenha uma moldura retangular");
    if (getComputedStyle(cartridgeCover).overflow !== "visible")
      throw Error("O brilho ainda é cortado nas bordas PNG pelo overflow do cartucho");
    if (!getComputedStyle(cartridgeCover.querySelector("img")).filter.includes("drop-shadow"))
      throw Error("O cartucho selecionado não tem o halo de cor da Hero");
    // Serialized writes must persist beyond the current fullscreen session.
    await wait(() => document.querySelector(".bp-settings-list"), "Configurações visíveis");
    let savedBp = await api.BigPictureOptions();
    const allSaved = () => savedBp.view === "cartridge" &&
      savedBp.background === "solid" &&
      savedBp.showTitle === false && savedBp.showPlatform === false &&
      savedBp.buttonStyle === "playstation";
    for (let retry = 0; retry < 60 && !allSaved(); retry++) {
      await new Promise((r) => setTimeout(r, 100));
      savedBp = await api.BigPictureOptions();
    }
    if (!allSaved())
      throw Error("Preferências do Big Picture não persistiram: " + JSON.stringify(savedBp));
    secondButtons[1].pressed = true;
    await wait(() => document.querySelector(".bp-settings").hidden,
      "Botão B do segundo controle fecha apenas configurações");
    secondButtons[1].pressed = false;
    await new Promise((r) => setTimeout(r, 120));
    secondButtons[1].pressed = true;
    await wait(() => !document.querySelector(".big-picture"), "Botão B fecha Big Picture");
    secondButtons[1].pressed = false;
    await probe.Progress("bigpicture-reopen");
    click("Big Picture");
    await wait(() => document.querySelector(".big-picture")?.classList.contains("bp-cartridges"),
      "Configuração persistida ao reabrir");
    if (!document.querySelector(".big-picture").classList.contains("bp-hide-title"))
      throw Error("Big Picture reabriu com títulos visíveis");
    if (document.querySelector(".big-picture").dataset.controller !== "playstation")
      throw Error("Padrão de botões PlayStation não foi recuperado ao abrir Big Picture");
    document.dispatchEvent(new KeyboardEvent("keydown", {key:"Escape",bubbles:true,cancelable:true}));
    await wait(() => !document.querySelector(".big-picture"), "Fechar modo Big Picture");
    api.Run = originalRun;
    await probe.Progress("all-passed");
    await probe.Done("");
  } catch (e) {
    await probe.Done(String(e.stack || e));
  }
})();
