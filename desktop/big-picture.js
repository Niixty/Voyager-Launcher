"use strict";

function openBigPicture() {
  if (document.querySelector(".big-picture")) return;
  const previous = document.activeElement;
  const panel = el("section", null, "big-picture");
  panel.setAttribute("aria-label", "Big Picture");
  const exit = btn("Sair · B", close);
  const grid = el("div", null, "bp-grid");
  const status = el("p", "", "bp-status");
  status.setAttribute("role", "status");
  const rows = [...games].sort((a, b) =>
    (a.Config.title || a.ID).localeCompare(b.Config.title || b.ID, "pt-BR"));
  const cards = [];
  let selected = 0, frame = 0, held = "", repeatAt = 0, busy = false;
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
  panel.append(add(el("header"), el("h1", "Big Picture"), exit), grid,
    add(el("footer"), status, el("span", "Direcional ou analógico para navegar   ·   A para jogar   ·   B para sair")));
  document.body.append(panel);
  main.inert = document.querySelector("aside").inert = true;
  window.runtime.WindowFullscreen();
  if (cards.length) select(0); else exit.focus();
  document.addEventListener("keydown", key, true);
  frame = requestAnimationFrame(poll);

  function select(i, focus = true) {
    cards[selected]?.classList.remove("selected");
    selected = i;
    cards[i].classList.add("selected");
    if (focus) cards[i].focus({preventScroll:true});
    cards[i].scrollIntoView({block:"nearest"});
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
    if (action === "back") return close();
    if (!cards.length) return;
    if (action === "play") return start(selected);
    const columns = getComputedStyle(grid).gridTemplateColumns.split(" ").length;
    const step = {left:-1, right:1, up:-columns, down:columns}[action];
    if (step) select(Math.max(0, Math.min(cards.length - 1, selected + step)));
  }
  function key(e) {
    const action = {ArrowLeft:"left", ArrowRight:"right", ArrowUp:"up",
      ArrowDown:"down", Escape:"back", Enter:"play"}[e.key];
    if (!action || (e.key === "Enter" && document.activeElement === exit)) return;
    e.preventDefault();
    if (!e.repeat || action !== "play") act(action);
  }
  function poll(now) {
    if (!panel.isConnected) return;
    frame = requestAnimationFrame(poll);
    if (!document.hasFocus()) { held = "inactive"; return; }
    let action = "";
    for (const pad of navigator.getGamepads?.() || []) {
      if (!pad?.connected || pad.mapping !== "standard") continue;
      const pressed = i => pad.buttons[i]?.pressed;
      if (pressed(1)) action = "back";
      else if (pressed(0)) action = "play";
      else if (pressed(12) || pad.axes[1] < -.55) action = "up";
      else if (pressed(13) || pad.axes[1] > .55) action = "down";
      else if (pressed(14) || pad.axes[0] < -.55) action = "left";
      else if (pressed(15) || pad.axes[0] > .55) action = "right";
      if (action) break;
    }
    if (!action) { held = ""; return; }
    if (held === "inactive") return;
    if (held !== action) { held = action; repeatAt = now + 390; act(action); }
    else if (action !== "play" && action !== "back" && now >= repeatAt) {
      repeatAt = now + 155;
      act(action);
    }
  }
}
