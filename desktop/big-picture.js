"use strict";

// Fullscreen library and settings share one controller-driven focus system.
// Gamepad state is tracked PER device: a connected idle gamepad must never
// block actions from another Xbox/PlayStation controller.
async function openBigPicture() {
  if (document.querySelector(".big-picture")) return;
  const defaults = {
    view: "grid", background: "hero", showTitle: true,
    showPlatform: true, blurBackground: true, dim: 65,
  };
  const prefs = { ...defaults, ...(await api.BigPictureOptions()) };
  const iconPack = window.VoyagerGamepadIcons;
  const previous = document.activeElement;
  const panel = el("section", null, "big-picture");
  panel.setAttribute("aria-label", "Big Picture");
  const background = el("div", null, "bp-hero-background");
  const logo = el("img", null, "bp-logo");
  logo.src = "/icon.ico";
  logo.alt = "Voyager";
  const heading = add(el("div", null, "bp-heading"), logo, add(
    el("div"), el("small", "BIBLIOTECA DE JOGOS"), el("h1", "Big Picture"),
  ));
  const familyLabel = el("small", "", "bp-family");
  const settingsButton = btn("⚙ Configurações", () => showSettings(true), "bp-open-settings");
  const exit = btn("", close, "bp-exit");
  exit.setAttribute("aria-label", "Sair do Big Picture");
  const headerActions = add(
    el("div", null, "bp-header-actions"), familyLabel, settingsButton, exit,
  );
  const grid = el("div", null, "bp-grid");
  const status = el("p", "", "bp-status");
  status.setAttribute("role", "status");
  const hints = el("div", null, "bp-controls");
  const footer = add(el("footer"), status, hints);
  const settingsView = el("section", null, "bp-settings");
  settingsView.setAttribute("aria-label", "Configurações do Big Picture");
  settingsView.hidden = true;
  const settingsSurface = el("div", null, "bp-settings-surface");
  const settingsHeader = add(el("div", null, "bp-settings-header"),
    el("h2", "Configurações do Big Picture"),
    el("p", "Use o direcional para navegar, A para alterar e B para voltar."),
  );
  const settingsList = el("div", null, "bp-settings-list");
  settingsSurface.append(settingsHeader, settingsList);
  settingsView.append(settingsSurface);
  const rows = [...games].sort((a, b) =>
    (a.Config.title || a.ID).localeCompare(b.Config.title || b.ID, "pt-BR"));
  const cards = [];
  const covers = [];
  const titles = [];
  const platforms = [];
  let selected = 0, frame = 0, busy = false;
  let settingsOpen = false, settingsIndex = 0;
  let currentFamily = "", saveChain = Promise.resolve();
  const heldByPad = new Map();

  for (const [i, game] of rows.entries()) {
    const title = game.Config.title || game.ID;
    const card = btn("", () => start(i), "bp-card");
    const cover = el("div", title.slice(0, 1), "bp-cover");
    const name = el("strong", title, "bp-card-title");
    const platform = el("small", game.Config.platform || "", "bp-card-platform");
    card.append(cover, name, platform);
    card.onfocus = () => { if (!settingsOpen) select(i, false); };
    grid.append(card);
    cards.push(card);
    covers.push(cover);
    titles.push(name);
    platforms.push(platform);
  }
  if (!rows.length) grid.append(el("p", "Adicione jogos na biblioteca para começar."));

  const optionSpecs = [
    ["view", "Imagens dos jogos", () => prefs.view === "cartridge" ? "Cartuchos" : "Capas Grid",
      () => { prefs.view = prefs.view === "grid" ? "cartridge" : "grid"; }],
    ["background", "Fundo", () => prefs.background === "hero" ? "Hero do jogo selecionado" : "Fundo padrão",
      () => { prefs.background = prefs.background === "hero" ? "solid" : "hero"; }],
    ["blurBackground", "Desfoque do fundo", () => prefs.blurBackground ? "Ativado" : "Desativado",
      () => { prefs.blurBackground = !prefs.blurBackground; }],
    ["showTitle", "Nome dos jogos", () => prefs.showTitle ? "Visível" : "Oculto",
      () => { prefs.showTitle = !prefs.showTitle; }],
    ["showPlatform", "Plataforma dos jogos", () => prefs.showPlatform ? "Visível" : "Oculta",
      () => { prefs.showPlatform = !prefs.showPlatform; }],
    ["dim", "Escurecimento do fundo", () => prefs.dim + "%",
      () => { prefs.dim = prefs.dim >= 85 ? 25 : prefs.dim + 15; }],
  ];
  const settingsButtons = optionSpecs.map(([key, name, value, change], index) => {
    const button = btn("", () => {
      change();
      renderPreferences();
      queueSave();
      button.focus({ preventScroll: true });
      settingsIndex = index;
    }, "bp-setting");
    button.dataset.setting = key;
    button.onfocus = () => { settingsIndex = index; };
    settingsList.append(button);
    return button;
  });
  const settingsBack = btn("Voltar para os jogos", () => showSettings(false), "bp-settings-back");
  settingsBack.onfocus = () => { settingsIndex = settingsButtons.length - 1; };
  settingsButtons.push(settingsBack);
  settingsList.append(settingsBack);

  panel.append(background, add(el("header", null, "bp-topbar"), heading, headerActions),
    grid, footer, settingsView);
  document.body.append(panel);
  main.inert = document.querySelector("aside").inert = true;
  window.runtime.WindowFullscreen();
  renderPreferences();
  if (cards.length) select(0); else settingsButton.focus();
  refreshGlyphs("xbox");
  document.addEventListener("keydown", key, true);
  frame = requestAnimationFrame(poll);

  function queueSave() {
    const snapshot = { ...prefs };
    const task = saveChain.catch(() => {}).then(() => api.SaveBigPictureOptions(snapshot));
    saveChain = task;
    safe(async () => await task);
  }
  function renderPreferences() {
    panel.dataset.view = prefs.view;
    panel.classList.toggle("bp-cartridges", prefs.view === "cartridge");
    panel.classList.toggle("bp-hide-title", !prefs.showTitle);
    panel.classList.toggle("bp-hide-platform", !prefs.showPlatform);
    panel.classList.toggle("bp-blur-hero", prefs.blurBackground);
    panel.style.setProperty("--bp-dim", String(prefs.dim / 100));
    for (let i = 0; i < covers.length; i++) {
      const game = rows[i], cover = covers[i];
      const kind = prefs.view === "cartridge" ? "cartridge" : "grid";
      const img = image(game.ID, kind);
      if (kind === "cartridge") {
        img.onerror = () => {
          if (img.isConnected) img.replaceWith(image(game.ID, "grid"));
        };
      }
      cover.replaceChildren(el("span", (game.Config.title || game.ID).slice(0, 1)), img);
    }
    optionSpecs.forEach(([key, label, value], i) => {
      settingsButtons[i].replaceChildren(
        el("span", label), el("strong", value()), el("span", "Alterar", "bp-setting-arrow"),
      );
    });
    refreshBackground();
  }
  function refreshBackground() {
    background.replaceChildren();
    if (prefs.background !== "hero" || !rows[selected]) return;
    const art = image(rows[selected].ID, "hero");
    art.alt = "";
    art.onerror = () => art.remove();
    background.append(art);
  }
  function refreshGlyphs(family) {
    if (family === currentFamily) return;
    currentFamily = family;
    panel.dataset.controller = family;
    familyLabel.textContent = family === "playstation"
      ? "Controle PlayStation" : "Controle Xbox / genérico";
    exit.replaceChildren(
      iconPack?.button(1, family, "") || el("span", "B"), el("span", "Sair"),
    );
    const hint = (icons, label) => {
      const wrapper = el("span", null, "bp-hint");
      const artwork = el("span", null, "bp-hint-icons");
      artwork.append(...icons.filter(Boolean));
      wrapper.append(artwork, el("span", label));
      return wrapper;
    };
    hints.replaceChildren(
      hint([iconPack?.icon("generic_dpad_up"), iconPack?.stick("left")], "Navegar"),
      hint([iconPack?.button(0, family)], "Selecionar / jogar"),
      hint([iconPack?.button(1, family)], "Voltar"),
      hint([iconPack?.button(9, family)], "Configurações"),
    );
  }
  function select(index, focus = true) {
    if (!cards[index]) return;
    cards[selected]?.classList.remove("selected");
    selected = index;
    cards[index].classList.add("selected");
    if (focus) cards[index].focus({ preventScroll: true });
    cards[index].scrollIntoView({ block: "nearest" });
    refreshBackground();
  }
  function showSettings(open) {
    settingsOpen = open;
    settingsView.hidden = !open;
    grid.inert = open;
    settingsButton.inert = open;
    exit.inert = open;
    if (open) settingsButtons[settingsIndex]?.focus({ preventScroll: true });
    else if (cards[selected]) cards[selected].focus({ preventScroll: true });
    else settingsButton.focus({ preventScroll: true });
  }
  function close() {
    cancelAnimationFrame(frame);
    document.removeEventListener("keydown", key, true);
    panel.remove();
    main.inert = document.querySelector("aside").inert = false;
    window.runtime.WindowUnfullscreen();
    previous?.focus();
  }
  async function start(i) {
    if (busy || !rows[i]) return;
    const game = rows[i];
    if (game.Problem) { status.textContent = game.Problem; return; }
    busy = true;
    status.textContent = "Abrindo " + (game.Config.title || game.ID) + "…";
    try {
      await api.Run(game.ID, "--game");
    } catch (e) {
      status.textContent = String(e);
    } finally {
      setTimeout(() => {
        busy = false;
        if (status.textContent.startsWith("Abrindo ")) status.textContent = "";
      }, 1000);
    }
  }
  function act(action) {
    if (action === "settings") { showSettings(!settingsOpen); return; }
    if (action === "back") {
      if (settingsOpen) showSettings(false);
      else close();
      return;
    }
    if (settingsOpen) {
      if (action === "play") {
        settingsButtons[settingsIndex]?.click();
      } else {
        const move = { up: -1, left: -1, down: 1, right: 1 }[action] || 0;
        settingsIndex = Math.max(0, Math.min(settingsButtons.length - 1, settingsIndex + move));
        settingsButtons[settingsIndex]?.focus({ preventScroll: true });
      }
      return;
    }
    if (!cards.length) return;
    if (action === "play") { start(selected); return; }
    const columns = Math.max(1, getComputedStyle(grid).gridTemplateColumns.split(" ").length);
    const step = { left: -1, right: 1, up: -columns, down: columns }[action] || 0;
    if (step) select(Math.max(0, Math.min(cards.length - 1, selected + step)));
  }
  function key(e) {
    const action = {
      ArrowLeft: "left", ArrowRight: "right", ArrowUp: "up", ArrowDown: "down",
      Escape: "back", Enter: "play", " ": "play", F1: "settings",
      s: "settings", S: "settings",
    }[e.key];
    if (!action) return;
    e.preventDefault();
    if (action === "play" && !settingsOpen && document.activeElement === exit) {
      close();
      return;
    }
    if (action === "play" && !settingsOpen && document.activeElement === settingsButton) {
      showSettings(true);
      return;
    }
    if (!e.repeat || !["play", "back", "settings"].includes(action)) act(action);
  }
  function poll(now) {
    if (!panel.isConnected) return;
    frame = requestAnimationFrame(poll);
    const pads = navigator.getGamepads?.() || [];
    let candidate = "";
    let family = "";
    let defaultFamily = "";
    const seen = new Set();
    for (let index = 0; index < pads.length; index++) {
      const pad = pads[index];
      if (!pad?.connected || (pad.mapping !== "standard" && pad.buttons?.length < 16)) continue;
      seen.add(index);
      if (!defaultFamily) defaultFamily = iconPack?.detect(pad) || "xbox";
      const pressed = (i) => !!pad.buttons?.[i]?.pressed;
      let action = "";
      if (pressed(9)) action = "settings";
      else if (pressed(1)) action = "back";
      else if (pressed(0)) action = "play";
      else if (pressed(12) || pad.axes?.[1] < -.55) action = "up";
      else if (pressed(13) || pad.axes?.[1] > .55) action = "down";
      else if (pressed(14) || pad.axes?.[0] < -.55) action = "left";
      else if (pressed(15) || pad.axes?.[0] > .55) action = "right";
      if (!action) { heldByPad.delete(index); continue; }
      const last = heldByPad.get(index);
      const directional = ["left", "right", "up", "down"].includes(action);
      if (!last || last.action !== action) {
        heldByPad.set(index, { action, repeatAt: now + 390 });
        if (!candidate) { candidate = action; family = iconPack?.detect(pad) || "xbox"; }
      } else if (directional && now >= last.repeatAt) {
        last.repeatAt = now + 155;
        if (!candidate) { candidate = action; family = iconPack?.detect(pad) || "xbox"; }
      }
    }
    for (const index of heldByPad.keys()) {
      if (!seen.has(index)) heldByPad.delete(index);
    }
    if (candidate) {
      refreshGlyphs(family);
      act(candidate);
    } else if (defaultFamily && heldByPad.size === 0) {
      refreshGlyphs(defaultFamily);
    }
  }
}
