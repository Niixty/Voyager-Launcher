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
    click("Editor de loading");
    await wait(
      () =>
        document
          .querySelector(".stage")
          ?.style.backgroundImage.includes("data:image/png"),
      "Composição nativa",
    );
    const layer = [...document.querySelectorAll(".panel select")].find((select) =>
      [...select.options].some((option) => option.value === "title"),
    );
    layer.value = "title";
    layer.dispatchEvent(new Event("change", { bubbles: true }));
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
    set(props, "240");
    props.dispatchEvent(new Event("change"));
    click("Desfazer");
    click("Refazer");
    click("Salvar layout");
    await wait(
      () => document.querySelector("#notice").textContent === "Layout salvo",
      "Salvar layout",
    );
    const saved = await api.Layout("Smoke");
    const titleSaved = saved.elements.find((n) => n.id === "title");
    if (titleSaved.textAlign !== "right" || titleSaved.verticalAlign !== "bottom") {
      throw Error("Alinhamento não persistido no layout");
    }
    await probe.Capture("editor");
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
    await probe.Done("");
  } catch (e) {
    await probe.Done(String(e.stack || e));
  }
})();
