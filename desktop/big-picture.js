"use strict";

// Fullscreen library and settings share one controller-driven focus system.
// Gamepad state is tracked PER device: a connected idle gamepad must never
// block actions from another Xbox/PlayStation controller.
async function openBigPicture() {
  if (document.querySelector(".big-picture")) return;
  const defaults = {
    view: "grid", background: "hero", buttonStyle: "auto", showTitle: true,
    showPlatform: true, blurBackground: true, dim: 65,
    scrollDirection: "vertical", columns: 4, rows: 2, iconSize: 100,
  };
  const prefs = { ...defaults, ...(await api.BigPictureOptions()) };
  if (!["auto", "xbox", "playstation"].includes(prefs.buttonStyle)) prefs.buttonStyle = "auto";
  if (!["vertical", "horizontal"].includes(prefs.scrollDirection)) prefs.scrollDirection = "vertical";
  prefs.columns = Number.isInteger(prefs.columns) ? Math.max(1, Math.min(8, prefs.columns)) : 4;
  prefs.rows = Number.isInteger(prefs.rows) ? Math.max(1, Math.min(4, prefs.rows)) : 2;
  prefs.iconSize = Number.isFinite(prefs.iconSize)
    ? Math.max(60, Math.min(140, Math.round(prefs.iconSize / 10) * 10)) : 100;
  const iconPack = window.VoyagerGamepadIcons;
  const previous = document.activeElement;
  const panel = el("section", null, "big-picture");
  panel.setAttribute("aria-label", "Big Picture");
  const background = el("div", null, "bp-hero-background");
  const logo = el("img", null, "bp-logo");
  logo.src = "/icon.ico";
  logo.alt = "Voyager";
  const platformHeading = el("h1", "Biblioteca de jogos");
  const heading = add(el("div", null, "bp-heading"), logo, add(
    el("div"), el("small", "BIBLIOTECA DE JOGOS"), platformHeading,
  ));
  const clock = el("time", "", "bp-clock");
  clock.setAttribute("aria-label", "Horário local");
  const clockDivider = el("span", "|", "bp-clock-divider");
  clockDivider.setAttribute("aria-hidden", "true");
  const settingsButton = btn("", () => showSettings(true), "bp-open-settings");
  settingsButton.setAttribute("aria-label", "Configurações (Select)");
  const exit = btn("", close, "bp-exit");
  exit.setAttribute("aria-label", "Sair do Big Picture");
  const headerActions = add(
    el("div", null, "bp-header-actions"), clock, clockDivider, settingsButton, exit,
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
  let selected = 0, frame = 0, busy = false, closing = false;
  let settingsOpen = false, settingsIndex = 0;
  let currentFamily = "", renderedButtonStyle = "", lastDetectedFamily = "xbox", activePadIndex = null, saveChain = Promise.resolve();
  const accentCache = new Map();
  let accentRequestSerial = 0;
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
    ["buttonStyle", "Padrão dos botões", () => ({
        auto: "Automático", xbox: "Xbox (A / B / X / Y)", playstation: "PlayStation (✕ / ○ / □ / △)",
      })[prefs.buttonStyle],
      () => {
        const styles = ["auto", "xbox", "playstation"];
        prefs.buttonStyle = styles[(styles.indexOf(prefs.buttonStyle) + 1) % styles.length];
      }],
    ["scrollDirection", "Direção da rolagem", () => prefs.scrollDirection === "horizontal" ? "Horizontal" : "Vertical",
      () => { prefs.scrollDirection = prefs.scrollDirection === "vertical" ? "horizontal" : "vertical"; }],
    ["columns", "Colunas visíveis", () => String(prefs.columns),
      () => { prefs.columns = prefs.columns >= 8 ? 1 : prefs.columns + 1; }],
    ["rows", "Linhas visíveis", () => String(prefs.rows),
      () => { prefs.rows = prefs.rows >= 4 ? 1 : prefs.rows + 1; }],
    ["iconSize", "Tamanho dos ícones", () => prefs.iconSize + "%",
      () => { prefs.iconSize = prefs.iconSize >= 140 ? 60 : prefs.iconSize + 10; }],
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
  updateClock();
  const clockInterval = window.setInterval(updateClock, 10000);
  main.inert = document.querySelector("aside").inert = true;
  window.runtime.WindowFullscreen();
  renderPreferences();
  if (cards.length) select(0); else settingsButton.focus();
  refreshGlyphs("xbox");
  document.addEventListener("keydown", key, true);
  // In horizontal mode a regular vertical mouse-wheel gesture scrolls sideways.
  grid.addEventListener("wheel", (event) => {
    if (prefs.scrollDirection !== "horizontal" || settingsOpen ||
        Math.abs(event.deltaY) <= Math.abs(event.deltaX) ||
        grid.scrollWidth <= grid.clientWidth) return;
    grid.scrollLeft += event.deltaY;
    event.preventDefault();
  }, { passive: false });
  frame = requestAnimationFrame(poll);

  function updateClock() {
    const now = new Date();
    clock.textContent = String(now.getHours()).padStart(2, "0") + ":" +
      String(now.getMinutes()).padStart(2, "0");
  }
  function queueSave() {
    const snapshot = { ...prefs };
    const task = saveChain.catch(() => {}).then(() => api.SaveBigPictureOptions(snapshot));
    saveChain = task;
    safe(async () => await task);
  }
  function renderPreferences() {
    panel.dataset.view = prefs.view;
    panel.classList.toggle("bp-cartridges", prefs.view === "cartridge");
    panel.classList.toggle("bp-scroll-horizontal", prefs.scrollDirection === "horizontal");
    grid.style.setProperty("--bp-columns", String(prefs.columns));
    grid.style.setProperty("--bp-rows", String(prefs.rows));
    grid.style.setProperty("--bp-icon-scale", String(prefs.iconSize / 100));
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
      // Changing rows/columns/size should not reload every image and flicker.
      if (cover.dataset.kind !== kind) {
        cover.replaceChildren(el("span", (game.Config.title || game.ID).slice(0, 1)), img);
        cover.dataset.kind = kind;
      }
    }
    optionSpecs.forEach(([key, label, value], i) => {
      settingsButtons[i].replaceChildren(
        el("span", label), el("strong", value()), el("span", "Alterar", "bp-setting-arrow"),
      );
    });
    if (cards[selected]) {
      cards[selected].scrollIntoView({ block: "nearest", inline: "nearest" });
    }
    refreshBackground();
    refreshGlyphs(lastDetectedFamily);
  }
  function refreshBackground() {
    background.replaceChildren();
    if (prefs.background !== "hero" || !rows[selected]) return;
    const art = image(rows[selected].ID, "hero");
    art.alt = "";
    art.onerror = () => art.remove();
    background.append(art);
  }
  function refreshGlyphs(detectedFamily) {
    lastDetectedFamily = detectedFamily || "xbox";
    const family = prefs.buttonStyle === "auto" ? lastDetectedFamily : prefs.buttonStyle;
    if (family === currentFamily && renderedButtonStyle === prefs.buttonStyle) return;
    currentFamily = family;
    renderedButtonStyle = prefs.buttonStyle;
    panel.dataset.controller = family;
    exit.replaceChildren(
      iconPack?.button(1, family, "") || el("span", "B"), el("span", "Sair"),
    );
    settingsButton.replaceChildren(
      iconPack?.button(8, family, "Select") || el("span", "Select"),
      el("span", "Configurações"),
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
      hint([iconPack?.button(8, family)], "Configurações"),
    );
  }
  // Use the same native heroaccent algorithm as loading title and cartridge.
  // The first visit requests the dominant color once per game; stale results
  // cannot recolor a different cartridge when a controller scrolls quickly.
  function refreshAccent() {
    const id = rows[selected]?.ID;
    const serial = ++accentRequestSerial;
    const apply = (value) => {
      if (serial !== accentRequestSerial || !panel.isConnected) return;
      const color = /^#[0-9a-f]{6}$/i.test(value || "") ? value.toUpperCase() : "#ACB7C1";
      panel.style.setProperty("--bp-glow-color", color);
      panel.style.setProperty("--bp-glow-soft", color + "88");
      panel.style.setProperty("--bp-glow-wide", color + "50");
    };
    if (!id) { apply("#ACB7C1"); return; }
    if (accentCache.has(id)) { apply(accentCache.get(id)); return; }
    // Never show the prior game's color while this Hero is loading.
    apply("#ACB7C1");
    api.BigPictureAccent(id).then((color) => {
      accentCache.set(id,color);
      apply(color);
    }).catch(() => apply("#ACB7C1"));
  }
  function select(index, focus = true) {
    if (!cards[index]) return;
    cards[selected]?.classList.remove("selected");
    selected = index;
    platformHeading.textContent = rows[index]?.Config?.platform?.trim() || "Biblioteca de jogos";
    cards[index].classList.add("selected");
    if (focus) cards[index].focus({ preventScroll: true });
    cards[index].scrollIntoView({ block: "nearest", inline: "nearest" });
    refreshBackground();
    refreshAccent();
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
  async function close() {
    if (closing) return;
    closing = true;
    // When B closes fullscreen immediately after customization, guarantee
    // every queued preference write has finished before leaving the view.
    await saveChain.catch(() => {});
    cancelAnimationFrame(frame);
    clearInterval(clockInterval);
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
    if (closing) return;
    if (action === "settings") { showSettings(!settingsOpen); return; }
    if (action === "back") {
      if (settingsOpen) showSettings(false);
      else close();
      return;
    }
    if (settingsOpen) {
      if (action === "play") {
        // Standard button 0 (Xbox A / PlayStation Cross) applies the
        // focused preference; this also works for a second controller.
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
    // The browser places games row-major vertically and column-major horizontally.
    const horizontal = prefs.scrollDirection === "horizontal";
    const track = horizontal ? prefs.rows : prefs.columns;
    const cross = selected % track;
    let next = selected;
    if (horizontal) {
      if (action === "left") next -= track;
      if (action === "right") next += track;
      if (action === "up" && cross > 0) next--;
      if (action === "down" && cross < track - 1) next++;
    } else {
      if (action === "up") next -= track;
      if (action === "down") next += track;
      if (action === "left" && cross > 0) next--;
      if (action === "right" && cross < track - 1) next++;
    }
    if (next >= 0 && next < cards.length && next !== selected) select(next);
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
    let candidateIndex = null;
    let defaultFamily = "";
    const seen = new Set();
    for (let index = 0; index < pads.length; index++) {
      const pad = pads[index];
      if (!pad?.connected || (pad.mapping !== "standard" && pad.buttons?.length < 16)) continue;
      seen.add(index);
      if (!defaultFamily) defaultFamily = iconPack?.detect(pad) || "xbox";
      const pressed = (i) => !!pad.buttons?.[i]?.pressed;
      let action = "";
      // Standard Gamepad API: 8 = Select/Back, 9 = Start/Menu.
      // Settings must be opened by Select; leave Start unused for future actions.
      if (pressed(8)) action = "settings";
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
        if (!candidate) { candidate = action; candidateIndex = index; family = iconPack?.detect(pad) || "xbox"; }
      } else if (directional && now >= last.repeatAt) {
        last.repeatAt = now + 155;
        if (!candidate) { candidate = action; candidateIndex = index; family = iconPack?.detect(pad) || "xbox"; }
      }
    }
    for (const index of heldByPad.keys()) {
      if (!seen.has(index)) heldByPad.delete(index);
    }
    if (candidate) {
      activePadIndex = candidateIndex;
      refreshGlyphs(family);
      act(candidate);
    } else if (activePadIndex !== null && seen.has(activePadIndex)) {
      // Keep the last controller's icons even with controller 0 idle.
      refreshGlyphs(iconPack?.detect(pads[activePadIndex]) || "xbox");
    } else if (defaultFamily && heldByPad.size === 0) {
      activePadIndex = null;
      refreshGlyphs(defaultFamily);
    }
  }
}
