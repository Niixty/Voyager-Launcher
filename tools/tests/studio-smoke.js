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
    const iconAPI = window.VoyagerGamepadIcons;
    if (!iconAPI || iconAPI.names.length !== 38)
      throw Error("Registro completo de ícones genéricos/Xbox/PlayStation ausente");
    if (iconAPI.buttonName(0, "xbox") !== "xb_a" || iconAPI.buttonName(0, "playstation") !== "ps_x")
      throw Error("Mapeamento dos botões Xbox/PlayStation está incorreto");
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
    buttons[15].pressed = false;
    buttons[0].pressed = true;
    await wait(() => launches.length > 0, "Botão A");
    await new Promise(r => setTimeout(r, 1150));
    if (launches.length !== 1 || launches[0][0] !== "Smoke2" || launches[0][1] !== "--game")
      throw Error("O botão A deve abrir uma vez o jogo selecionado");
    buttons[0].pressed = false;
    buttons[1].pressed = true;
    await wait(() => !document.querySelector(".big-picture"), "Botão B");
    api.Run = originalRun;
    await probe.Progress("all-passed");
    await probe.Done("");
  } catch (e) {
    await probe.Done(String(e.stack || e));
  }
})();
