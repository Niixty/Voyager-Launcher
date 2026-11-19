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
    click("Editor de loading");
    await wait(
      () =>
        document
          .querySelector(".stage")
          ?.style.backgroundImage.includes("data:image/png"),
      "Composição nativa",
    );
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
    await probe.Capture("editor");
    click("Configurações");
    await wait(() => heading("Configurações"), "Configurações");
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
