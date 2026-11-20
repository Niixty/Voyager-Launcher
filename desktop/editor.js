"use strict";
async function editor(g) {
  const serial = page,
    [initial, factory] = await Promise.all([api.Layout(g.ID), api.Factory()]);
  if (page !== serial) return;
  let doc = structuredClone(initial),
    sel = "cartridge",
    snap = true,
    grid = true,
    zoom = 1,
    anim = false,
    previewPercent = 50,
    finalPreviewAt = 0,
    rev = 0,
    busy = false,
    drag = null;
  const undo = [],
    redo = [],
    names = {
      cartridge: "Cartucho",
      platform: "Plataforma",
      title: "Título",
      description: "Descrição",
      heading: "Mensagem",
      percentage: "Porcentagem",
      progress: "Barra antiga (desativada)",
      phase: "Etapa",
    };
  const controls = row(),
    outer = el("div", null, "stageScroll"),
    stage = el("div", null, "stage"),
    props = el("section", null, "panel"),
    layer = el("select");
  const pick = () => doc.elements.find((x) => x.id === sel);
  function remember() {
    undo.push(structuredClone(doc));
    if (undo.length > 80) undo.shift();
    redo.length = 0;
    dirty = true;
  }
  function history(from, to) {
    if (!from.length) return;
    to.push(doc);
    doc = from.pop();
    dirty = true;
    refresh();
  }
  function check(text, val, fn) {
    const box = inp("", "checkbox");
    box.checked = val;
    box.onchange = () => fn(box.checked);
    return add(el("label"), box, document.createTextNode(text));
  }
  const zoomer = options(
    [
      ["1", "Ajustar à janela"],
      ["1.5", "Zoom 150%"],
      ["2", "Zoom 200%"],
    ],
    "1",
  );
  zoomer.onchange = () => {
    zoom = Number(zoomer.value);
    stage.style.width = zoom * 100 + "%";
  };
  const scrub = inp(50, "range");
  scrub.min = 0;
  scrub.max = 100;
  scrub.step = 5;
  const scrubLabel = el("span", "Progresso: 50%");
  scrub.oninput = () => {
    const previous = previewPercent;
    previewPercent = Number(scrub.value);
    scrubLabel.textContent = "Progresso: " + previewPercent + "%";
    finalPreviewAt = previewPercent === 100 && previous !== 100 ? performance.now() : (previewPercent !== 100 ? 0 : finalPreviewAt);
    requestFrame();
  };
  controls.append(
    check("Grade", grid, (v) => {
      grid = v;
      outlines();
    }),
    check("Snap 10 px", snap, (v) => (snap = v)),
    check("Animar cartucho", anim, (v) => {
      anim = v;
      refresh();
    }),
    zoomer,
    add(el("label", null, "previewProgress"), scrubLabel, scrub),
    btn("Desfazer", () => history(undo, redo)),
    btn("Refazer", () => history(redo, undo)),
  );
  main.append(controls, outer, props);
  outer.append(stage);
  stage.tabIndex = 0;
  stage.setAttribute("aria-label", "Área de edição, 1600 por 900 pixels");
  function limit(v, a, b) {
    return Math.max(a, Math.min(v, b));
  }
  function bounds(n) {
    n.width = limit(n.width, 16, 1600);
    n.height = limit(n.height, 6, 900);
    n.x = limit(n.x, 0, 1600 - n.width);
    n.y = limit(n.y, 0, 900 - n.height);
  }
  function position(node, n) {
    Object.assign(node.style, {
      left: n.x / 16 + "%",
      top: n.y / 9 + "%",
      width: n.width / 16 + "%",
      height: n.height / 9 + "%",
    });
  }
  let parts = null,
    floating = null,
    raf = 0;
  function outlines() {
    stage.replaceChildren();
    floating = null;
    if (anim && parts) {
      const n = doc.elements.find((x) => x.id === "cartridge");
      if (n.visible) {
        floating = el("img");
        floating.src = parts.sprite || image(g.ID, parts.kind).src;
        floating.className = "floating";
        floating.style.filter = previewPercent >= 95
          ? "drop-shadow(0 0 16px " + (parts.accent || "#afc9fa") + ")"
          : "none";
        position(floating, n);
        stage.append(floating);
      }
      const ink = el("img", null, "foreground");
      ink.src = parts.above;
      stage.append(ink);
    }
    if (grid) {
      const lines = el("div", null, "gridlines");
      for (let x = 100; x < 1600; x += 100) {
        const tick = el("small", String(x));
        tick.style.cssText =
          "position:absolute;top:0;font-size:9px;left:" + x / 16 + "%";
        lines.append(tick);
      }
      for (let y = 100; y < 900; y += 100) {
        const tick = el("small", String(y));
        tick.style.cssText =
          "position:absolute;left:0;font-size:9px;top:" + y / 9 + "%";
        lines.append(tick);
      }
      stage.append(lines);
    }
    for (const n of doc.elements) {
      if (!n.visible || n.id === "progress") continue;
      const box = el("div", null, "node" + (n.id === sel ? " selected" : ""));
      box.title = names[n.id];
      box.dataset.id = n.id;
      position(box, n);
      if (n.id === sel) {
        for (const [x, y] of [
          [0, 0],
          [1, 0],
          [0, 1],
          [1, 1],
        ]) {
          const h = el("i", null, "knob");
          h.style.left = x * 100 + "%";
          h.style.top = y * 100 + "%";
          h.style.cursor = x === y ? "nwse-resize" : "nesw-resize";
          h.dataset.corner = x + "," + y;
          box.append(h);
        }
      }
      stage.append(box);
    }
  }
  stage.onpointerdown = (e) => {
    const box = e.target.closest(".node");
    if (!box) return;
    e.preventDefault();
    stage.focus();
    sel = box.dataset.id;
    const r = stage.getBoundingClientRect();
    drag = {
      id: e.pointerId,
      sx: e.clientX,
      sy: e.clientY,
      w: r.width,
      h: r.height,
      original: { ...pick() },
      corner: e.target.dataset.corner,
      moved: false,
    };
    stage.setPointerCapture(e.pointerId);
    outlines();
    properties();
  };
  stage.onpointermove = (e) => {
    if (!drag || drag.id !== e.pointerId) return;
    const dx = ((e.clientX - drag.sx) * 1600) / drag.w,
      dy = ((e.clientY - drag.sy) * 900) / drag.h;
    if (Math.abs(dx) + Math.abs(dy) < 1 && !drag.moved) return;
    if (!drag.moved) {
      remember();
      drag.moved = true;
    }
    const o = drag.original,
      n = pick(),
      quant = (v) => (snap ? Math.round(v / 10) * 10 : Math.round(v));
    if (!drag.corner) {
      n.x = quant(o.x + dx);
      n.y = quant(o.y + dy);
    } else {
      const [right, bottom] = drag.corner.split(",").map(Number);
      const x = right ? o.x : limit(quant(o.x + dx), 0, o.x + o.width - 16),
        y = bottom ? o.y : limit(quant(o.y + dy), 0, o.y + o.height - 6);
      n.width = right ? quant(o.width + dx) : o.x + o.width - x;
      n.height = bottom ? quant(o.height + dy) : o.y + o.height - y;
      n.x = x;
      n.y = y;
    }
    bounds(n);
    position(stage.querySelector(".selected"), n);
    requestFrame();
  };
  stage.onpointerup = stage.onpointercancel = () => {
    drag = null;
    refresh();
  };
  function edit(fn) {
    remember();
    fn(pick());
    bounds(pick());
    refresh();
  }
  function properties() {
    props.replaceChildren();
    layer.replaceChildren();
    for (const n of [...doc.elements].reverse().filter((x) => x.id !== "progress")) {
      const o = el("option", names[n.id] + (n.visible ? "" : " (oculto)"));
      o.value = n.id;
      layer.append(o);
    }
    layer.value = sel;
    layer.onchange = () => {
      sel = layer.value;
      properties();
      outlines();
    };
    const n = pick(),
      fields = el("div", null, "fields");
    props.append(field("Camada selecionada", layer), fields);
    for (const [k, label] of [
      ["x", "X"],
      ["y", "Y"],
      ["width", "Largura"],
      ["height", "Altura"],
      ["fontSize", "Fonte"],
    ]) {
      if (k === "fontSize" && !n.fontSize) continue;
      const i = inp(Math.round(n[k]), "number");
      i.onchange = () =>
        edit((obj) => {
          if (i.value !== "")
            obj[k] = k === "fontSize" ? limit(+i.value, 9, 90) : +i.value;
        });
      fields.append(field(label, i));
    }
    const color = options(
      [
        ["auto", "Automática"],
        ["hero", "Hero"],
        ["white", "Branco"],
        ["muted", "Cinza"],
      ],
      n.color || "auto",
    );
    color.onchange = () => edit((obj) => (obj.color = color.value));
    if (n.fontSize) {
      fields.append(field("Cor", color));
      const textAlign = options(
        [
          ["left", "Esquerda"],
          ["center", "Centro"],
          ["right", "Direita"],
        ],
        n.textAlign || "left",
      );
      textAlign.dataset.textAlign = "horizontal";
      textAlign.onchange = () => edit((obj) => (obj.textAlign = textAlign.value));
      const verticalAlign = options(
        [
          ["top", "Topo"],
          ["middle", "Centro"],
          ["bottom", "Embaixo"],
        ],
        n.verticalAlign || "top",
      );
      verticalAlign.dataset.textAlign = "vertical";
      verticalAlign.onchange = () =>
        edit((obj) => (obj.verticalAlign = verticalAlign.value));
      fields.append(
        field("Alinhamento horizontal do texto", textAlign),
        field("Alinhamento vertical do texto", verticalAlign),
      );
    }
    const tools = row(
      check("Visível", n.visible, (v) => edit((obj) => (obj.visible = v))),
    );
    for (const [d, t] of [
      [-1, "Para trás"],
      [1, "Para frente"],
    ])
      tools.append(
        btn(t, () => {
          const i = doc.elements.indexOf(n),
            j = i + d;
          if (j >= 0 && j < doc.elements.length)
            edit(() => {
              [doc.elements[i], doc.elements[j]] = [
                doc.elements[j],
                doc.elements[i],
              ];
            });
        }),
      );
    const align = options(
      [
        ["", "Posicionar elemento…"],
        ["l", "Esquerda"],
        ["cx", "Centro horizontal"],
        ["r", "Direita"],
        ["t", "Topo"],
        ["cy", "Centro vertical"],
        ["b", "Base"],
      ],
      "",
    );
    align.onchange = () =>
      edit((o) => {
        const x = { l: 0, cx: (1600 - o.width) / 2, r: 1600 - o.width },
          y = { t: 0, cy: (900 - o.height) / 2, b: 900 - o.height };
        if (align.value in x) o.x = x[align.value];
        if (align.value in y) o.y = y[align.value];
      });
    tools.append(
      align,
      btn("Restaurar elemento", () =>
        edit((o) =>
          Object.assign(
            o,
            factory.elements.find((f) => f.id === sel),
          ),
        ),
      ),
    );
    props.append(tools);
    // A grade 3x3 move a caixa completa, sem alterar o alinhamento do texto.
    const anchors = el("div", null, "anchorPresets");
    for (const [label, name, horizontal, vertical] of [
      ["↖", "Topo esquerdo", 0, 0],
      ["↑", "Topo central", 0.5, 0],
      ["↗", "Topo direito", 1, 0],
      ["←", "Centro esquerdo", 0, 0.5],
      ["●", "Centro da tela", 0.5, 0.5],
      ["→", "Centro direito", 1, 0.5],
      ["↙", "Inferior esquerdo", 0, 1],
      ["↓", "Inferior central", 0.5, 1],
      ["↘", "Inferior direito", 1, 1],
    ]) {
      const button = btn(label, () =>
        edit((o) => {
          o.x = Math.round((1600 - o.width) * horizontal);
          o.y = Math.round((900 - o.height) * vertical);
        }),
      );
      button.title = "Posicionar: " + name;
      button.setAttribute("aria-label", "Posicionar: " + name);
      anchors.append(button);
    }
    const heading = el("div", "POSIÇÃO RÁPIDA NA TELA", "alignHeading");
    const hint = el("small", "Move o elemento inteiro. Para alinhar somente o texto, use os campos acima.");
    props.append(add(el("div", null, "alignQuick"), heading, hint, anchors));
  }
  let frameTimer;
  function requestFrame() {
    rev++;
    clearTimeout(frameTimer);
    frameTimer = setTimeout(() => safe(render), 100);
  }
  async function render() {
    if (busy || page !== serial) return;
    busy = true;
    const seq = rev;
    try {
      const layers = anim ? await api.AnimationLayersProgress(g.ID, doc, previewPercent) : null;
      const frame = layers ? layers.base : await api.FrameProgress(g.ID, doc, previewPercent);
      if (page === serial && seq === rev) {
        parts = layers;
        stage.style.backgroundImage = "url(" + frame + ")";
        outlines();
        if (anim && !raf) raf = requestAnimationFrame(float);
      }
    } finally {
      busy = false;
      if (rev !== seq && page === serial) safe(render);
    }
  }
  function refresh() {
    outlines();
    properties();
    requestFrame();
  }
  function keys(e) {
    if (
      page !== serial ||
      ["INPUT", "SELECT", "TEXTAREA"].includes(e.target.tagName)
    )
      return;
    if (e.ctrlKey && ["z", "y"].includes(e.key.toLowerCase())) {
      e.preventDefault();
      if (e.key.toLowerCase() === "z") history(undo, redo);
      else history(redo, undo);
    } else if (e.key.startsWith("Arrow")) {
      e.preventDefault();
      const v = e.shiftKey ? 10 : 1;
      edit((n) => {
        n.x += { ArrowLeft: -v, ArrowRight: v }[e.key] || 0;
        n.y += { ArrowUp: -v, ArrowDown: v }[e.key] || 0;
      });
    }
  }
  main.onkeydown = keys;
  function float(time) {
    raf = 0;
    if (page !== serial || !anim) return;
    if (floating) {
      const finishing = previewPercent >= 95;
      const elapsed = Math.max(0, time - finalPreviewAt);
      const jump = previewPercent === 100 && finalPreviewAt
        ? Math.sin(Math.PI * Math.min(1, elapsed / 440))
        : 0;
      const offset = finishing ? -8 * jump
        : (11 * Math.sin((time / 2900) * 2 * Math.PI) * stage.clientHeight) / 900;
      const scale = finishing ? (1.035 + 0.075 * jump) : 1;
      floating.style.transform = "translateY(" + offset + "px) scale(" + scale + ")";
    }
    raf = requestAnimationFrame(float);
  }
  const scope = options(
    [
      ["game", "Somente este jogo"],
      ["future", "Padrão para jogos sem layout próprio"],
      ["all", "Padrão para todos os jogos"],
    ],
    "game",
  );
  main.append(
    row(
      scope,
      btn(
        "Salvar layout",
        async () => {
          if (
            scope.value === "all" &&
            !confirm(
              "Aplicar a todos os jogos e guardar os layouts anteriores em backup?",
            )
          )
            return;
          await api.SaveLayout(g.ID, scope.value, doc);
          dirty = false;
          note("Layout salvo");
        },
        "primary",
      ),
      btn("Restaurar layout", () => {
        if (confirm("Restaurar os oito elementos de fábrica?")) {
          remember();
          doc = structuredClone(factory);
          refresh();
        }
      }),
      btn("Print do layout salvo (50%)", () => api.Run(g.ID, "--preview")),
      btn("Testar loading (10 s)", () => api.Run(g.ID, "--test-loading")),
    ),
  );
  refresh();
}
