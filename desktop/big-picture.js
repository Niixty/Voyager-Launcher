"use strict";

function openBigPicture() {
  if (document.querySelector(".big-picture")) return;

  const iconPack = window.VoyagerGamepadIcons;
  const previous = document.activeElement;
  const panel = el("section", null, "big-picture");
  panel.setAttribute("aria-label", "Big Picture");

  const logo = el("img", null, "bp-logo");
  logo.src = "/icon.ico";
  logo.alt = "Voyager";
  const heading = add(
    el("div", null, "bp-heading"),
    logo,
    add(
      el("div"),
      el("small", "MODO TELA CHEIA"),
      el("h1", "Big Picture"),
    ),
  );

  const familyLabel = el("small", "", "bp-family");
  const exit = btn("", close, "bp-exit");
  exit.setAttribute("aria-label", "Sair do Big Picture");
  const headerActions = add(el("div", null, "bp-header-actions"), familyLabel, exit);

  const grid = el("div", null, "bp-grid");
  const status = el("p", "", "bp-status");
  status.setAttribute("role", "status");
  const hints = el("div", null, "bp-controls");
  const footer = add(el("footer"), status, hints);
  const rows = [...games].sort((a, b) =>
    (a.Config.title || a.ID).localeCompare(b.Config.title || b.ID, "pt-BR"));
  const cards = [];

  let selected = 0;
  let frame = 0;
  let held = "";
  let repeatAt = 0;
  let busy = false;
  let currentFamily = "";

  for (const [i, game] of rows.entries()) {
    const card = btn("", () => start(i), "bp-card");
    const title = game.Config.title || game.ID;
    const cover = el("div", title.slice(0, 1), "bp-cover");
    cover.append(image(game.ID, "grid"));
    card.append(cover, el("strong", title));
    card.onfocus = () => select(i, false);
    cards.push(card);
    grid.append(card);
  }
  if (!rows.length) grid.append(el("p", "Adicione jogos na biblioteca para começar."));

  panel.append(add(el("header", null, "bp-topbar"), heading, headerActions), grid, footer);
  document.body.append(panel);
  main.inert = document.querySelector("aside").inert = true;
  window.runtime.WindowFullscreen();
  if (cards.length) select(0);
  else exit.focus();

  document.addEventListener("keydown", key, true);
  refreshGlyphs("xbox");
  frame = requestAnimationFrame(poll);

  function actionHint(iconNodes, description, extra = "") {
    const wrapper = el("span", null, "bp-hint");
    const artwork = el("span", null, "bp-hint-icons");
    for (const glyph of iconNodes) {
      if (glyph) artwork.append(glyph);
    }
    wrapper.append(artwork, el("span", description));
    if (extra) wrapper.title = extra;
    return wrapper;
  }
  function refreshGlyphs(family) {
    if (family === currentFamily) return;
    currentFamily = family;
    panel.dataset.controller = family;

    familyLabel.textContent = family === "playstation" ? "Controle PlayStation" : "Controle Xbox / genérico";
    exit.replaceChildren();
    exit.append(iconPack.button(1, family, "") || el("span", "B"), el("span", "Sair"));
    hints.replaceChildren(
      actionHint(
        [iconPack.icon("generic_dpad_up"), iconPack.stick("left")],
        "Navegar",
        "Direcional ou analógico esquerdo · Teclas de seta",
      ),
      actionHint(
        [iconPack.button(0, family)],
        "Iniciar jogo",
        "Botão inferior · Enter",
      ),
      actionHint(
        [iconPack.button(1, family)],
        "Voltar",
        "Botão direito · Esc",
      ),
    );
  }
  function select(i, focus = true) {
    cards[selected]?.classList.remove("selected");
    selected = i;
    cards[i].classList.add("selected");
    if (focus) cards[i].focus({ preventScroll: true });
    cards[i].scrollIntoView({ block: "nearest" });
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
    if (game.Problem) {
      status.textContent = game.Problem;
      return;
    }
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
    if (action === "back") return close();
    if (!cards.length) return;
    if (action === "play") return start(selected);
    const columns = Math.max(1, getComputedStyle(grid).gridTemplateColumns.split(" ").length);
    const step = { left: -1, right: 1, up: -columns, down: columns }[action];
    if (step) select(Math.max(0, Math.min(cards.length - 1, selected + step)));
  }
  function key(e) {
    const action = {
      ArrowLeft: "left",
      ArrowRight: "right",
      ArrowUp: "up",
      ArrowDown: "down",
      Escape: "back",
      Enter: "play",
    }[e.key];
    if (!action || (e.key === "Enter" && document.activeElement === exit)) return;
    e.preventDefault();
    if (!e.repeat || action !== "play") act(action);
  }
  function poll(now) {
    if (!panel.isConnected) return;
    frame = requestAnimationFrame(poll);
    if (!document.hasFocus()) {
      held = "inactive";
      return;
    }
    let action = "";
    const pads = navigator.getGamepads?.() || [];
    for (const pad of pads) {
      if (!pad?.connected || pad.mapping !== "standard") continue;
      refreshGlyphs(iconPack.detect(pad));
      const pressed = i => pad.buttons[i]?.pressed;
      if (pressed(1)) action = "back";
      else if (pressed(0)) action = "play";
      else if (pressed(12) || pad.axes?.[1] < -.55) action = "up";
      else if (pressed(13) || pad.axes?.[1] > .55) action = "down";
      else if (pressed(14) || pad.axes?.[0] < -.55) action = "left";
      else if (pressed(15) || pad.axes?.[0] > .55) action = "right";
      break; // Use the first connected standard controller for glyphs and navigation.
    }
    if (!action) {
      held = "";
      return;
    }
    if (held === "inactive") return;
    if (held !== action) {
      held = action;
      repeatAt = now + 390;
      act(action);
    } else if (action !== "play" && action !== "back" && now >= repeatAt) {
      repeatAt = now + 155;
      act(action);
    }
  }
}
