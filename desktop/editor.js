"use strict";
async function editor(g) {
  const serial = page,
    [initial, factory] = await Promise.all([api.Layout(g.ID), api.Factory()]);
  if (page !== serial) return;
  let doc = structuredClone(initial),
    sel = "cartridge",
    selected = new Set(["cartridge"]),
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
      percentage: "Porcentagem",
      progress: "Barra antiga (desativada)",
      phase: "Etapa",
    };
  const controls = row(),
    outer = el("div", null, "stageScroll"),
    stage = el("div", null, "stage"),
    props = el("section", null, "panel"),
    stageEditor = el("section", null, "panel stageMessageEditor"),
    layer = el("select");
  const pick = () => doc.elements.find((x) => x.id === sel);
  const selectedNodes = () => doc.elements.filter((x) => selected.has(x.id));
  // Align the selection's combined bounding box, preserving internal offsets.
  // No Math.round here: odd-size elements require half-pixel origins to
  // place their true center at 800 x 450 in a 1600 x 900 canvas.
  function positionSelection(horizontal, vertical) {
    const nodes = selectedNodes();
    if (!nodes.length) return;
    const x0 = Math.min(...nodes.map((n) => n.x));
    const x1 = Math.max(...nodes.map((n) => n.x + n.width));
    const y0 = Math.min(...nodes.map((n) => n.y));
    const y1 = Math.max(...nodes.map((n) => n.y + n.height));
    const dx = horizontal === null ? 0 : (1600 - (x1 - x0)) * horizontal - x0;
    const dy = vertical === null ? 0 : (900 - (y1 - y0)) * vertical - y0;
    if (!dx && !dy) return;
    remember();
    for (const node of nodes) {
      node.x += dx;
      node.y += dy;
      bounds(node);
    }
    refresh();
  }
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
  main.append(controls, outer, props, stageEditor);
  outer.append(stage);
  stage.tabIndex = 0;
  stage.setAttribute("aria-label", "Área de edição, 1600 por 900 pixels");
  function limit(v, a, b) {
    return Math.max(a, Math.min(v, b));
  }
  function bounds(n) {
    n.width = limit(n.width, 16, 1600);
    n.height = limit(n.height, 6, 900);
    // X/Y may deliberately extend beyond the virtual screen.
    n.x = limit(n.x, -3200, 3200);
    n.y = limit(n.y, -3200, 3200);
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
  let motionY = 0,
    motionScale = 1,
    lastAnimationFrame = 0;
  function outlines() {
    // Retain the animated DOM image to avoid restarting its transform.
    const previousFloating = floating;
    stage.replaceChildren();
    floating = null;
    if (anim && parts) {
      const n = doc.elements.find((x) => x.id === "cartridge");
      if (n.visible) {
        floating = previousFloating || el("img");
        const source = parts.sprite || image(g.ID, parts.kind).src;
        if (floating.getAttribute("src") !== source) floating.src = source;
        floating.className = "floating";
        const glow = Math.max(0, Math.min(1, previewPercent / 100));
        const accent = (parts.accent || "#afc9fa").replace("#", "").slice(0, 6);
        const alpha = Math.round((0.08 + 0.6 * glow ** 1.3) * 255).toString(16).padStart(2, "0");
        floating.style.filter = "drop-shadow(0 0 " + (4 + 14 * glow).toFixed(1) + "px #" + accent + alpha + ")";
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
      const box = el("div", null, "node" + (selected.has(n.id) ? " selected" : ""));
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
    if (!box) {
      // Clicking the background, even with Shift held, clears the entire
      // selection without changing the saved layout or dirty flag.
      drag = null;
      if (selected.size || sel) {
        selected.clear();
        sel = "";
        outlines();
        properties();
      }
      return;
    }
    e.preventDefault();
    stage.focus();
    const id = box.dataset.id;
    // Shift + click toggles elements without starting an accidental drag.
    if (e.shiftKey) {
      if (selected.has(id)) {
        selected.delete(id);
        if (sel === id) sel = [...selected].at(-1) || "";
      } else {
        selected.add(id);
        sel = id;
      }
      outlines();
      properties();
      return;
    }
    // Clicking a member of a multi-selection keeps the group together;
    // clicking outside it selects only the new element.
    if (!selected.has(id)) selected = new Set([id]);
    sel = id;
    const r = stage.getBoundingClientRect();
    drag = {
      id: e.pointerId,
      sx: e.clientX,
      sy: e.clientY,
      w: r.width,
      h: r.height,
      original: { ...pick() },
      originals: selectedNodes().map((n) => ({ id: n.id, x: n.x, y: n.y })),
      corner: e.target.dataset.corner,
      moved: false,
    };
    stage.setPointerCapture(e.pointerId);
    outlines();
    properties();
  };
  outer.addEventListener("pointerdown", (event) => {
    if (event.target !== outer) return;
    drag = null;
    if (selected.size || sel) {
      selected.clear();
      sel = "";
      outlines();
      properties();
    }
  });
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
      // Snap the common delta, not each original coordinate. This keeps
      // selected elements' exact relative spacing and center positions.
      const shiftX = quant(dx), shiftY = quant(dy);
      for (const entry of drag.originals) {
        const target = doc.elements.find((item) => item.id === entry.id);
        if (!target) continue;
        target.x = entry.x + shiftX;
        target.y = entry.y + shiftY;
        bounds(target);
      }
    } else {
      const [right, bottom] = drag.corner.split(",").map(Number);
      const x = right ? o.x : limit(quant(o.x + dx), -3200, o.x + o.width - 16),
        y = bottom ? o.y : limit(quant(o.y + dy), -3200, o.y + o.height - 6);
      n.width = right ? quant(o.width + dx) : o.x + o.width - x;
      n.height = bottom ? quant(o.height + dy) : o.y + o.height - y;
      n.x = x;
      n.y = y;
    }
    bounds(n);
    stage.querySelectorAll(".node.selected").forEach((box) => {
      const element = doc.elements.find((item) => item.id === box.dataset.id);
      if (element) position(box, element);
    });
    requestFrame();
  };
  stage.onpointerup = stage.onpointercancel = () => {
    drag = null;
    refresh();
  };
  function edit(fn) {
    if (!pick()) return;
    remember();
    fn(pick());
    bounds(pick());
    refresh();
  }
  function properties() {
    props.replaceChildren();
    layer.replaceChildren();
    const emptyOption = el("option", "Nenhuma camada selecionada");
    emptyOption.value = "";
    layer.append(emptyOption);
    for (const n of [...doc.elements].reverse().filter((x) => x.id !== "progress")) {
      const o = el("option", names[n.id] + (n.visible ? "" : " (oculto)"));
      o.value = n.id;
      layer.append(o);
    }
    layer.value = sel;
    layer.onchange = () => {
      sel = layer.value;
      selected = new Set([sel]);
      properties();
      outlines();
    };
    const n = pick(),
      fields = el("div", null, "fields");
    props.append(field("Camada selecionada", layer));
    const toggles = el("div", null, "layerToggles");
    toggles.append(el("strong", "Ativar ou desativar camadas"));
    for (const node of doc.elements.filter((item) => item.id !== "progress")) {
      const option = check(names[node.id], node.visible, (value) => {
        remember();
        node.visible = value;
        refresh();
      });
      option.dataset.layerId = node.id;
      toggles.append(option);
    }
    props.append(toggles);
    if (!n) {
      props.append(el("p", "Nenhuma camada selecionada. Clique em uma camada para editar ou marque uma camada acima para exibi-la.", "emptyLayerHint"));
      return;
    }
    props.append(fields);
    props.append(el("small", selected.size > 1
      ? selected.size + " camadas selecionadas — arraste qualquer uma para mover o grupo. Shift + clique para adicionar/remover."
      : "Shift + clique em outras camadas para mover várias juntas."));
    props.append(el("small", "Posicionamento livre: X e Y de -3200 a +3200. Valores negativos, como Y = -50, colocam parte do elemento fora da tela."));
    for (const [k, label] of [
      ["x", "X"],
      ["y", "Y"],
      ["width", "Largura"],
      ["height", "Altura"],
      ["fontSize", "Fonte"],
    ]) {
      if (k === "fontSize" && !n.fontSize) continue;
      const i = inp(n[k], "number");
      if (k === "x" || k === "y") {
        i.min = -3200;
        i.max = 3200;
        i.step = "any";
      }
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
      if (n.id === "title") {
        fields.append(el("small", "Cor automática da Hero: a mesma do preenchimento e brilho do Cartucho/Grid. Atualiza conforme o jogo."));
      } else {
        fields.append(field("Cor", color));
      }
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
      check("Camada ativada", n.visible, (v) => edit((obj) => (obj.visible = v))),
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
    align.onchange = () => {
      const positions = {
        l: [0, null], cx: [0.5, null], r: [1, null],
        t: [null, 0], cy: [null, 0.5], b: [null, 1],
      };
      if (positions[align.value]) positionSelection(...positions[align.value]);
    };
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
      const button = btn(label, () => positionSelection(horizontal, vertical));
      button.title = "Posicionar: " + name;
      button.setAttribute("aria-label", "Posicionar: " + name);
      anchors.append(button);
    }
    const heading = el("div", "POSIÇÃO RÁPIDA NA TELA", "alignHeading");
    const hint = el("small", "Alinha a caixa ou o conjunto selecionado, sem alterar o espaçamento interno. O centro é exato, mesmo com dimensões ímpares.");
    const axisButtons = el("div", null, "axisAlign");
    axisButtons.append(
      btn("Centralizar só horizontal", () => positionSelection(0.5, null)),
      btn("Centralizar só vertical", () => positionSelection(null, 0.5)),
      btn("Centralizar nos dois eixos", () => positionSelection(0.5, 0.5)),
    );
    props.append(add(el("div", null, "alignQuick"), heading, hint, axisButtons, anchors));
  }
  // The ten stage messages live in loading_layout.json so the existing
  // Save layout scope (game/future/all) also controls their inheritance.
  const stageDefinitions = [
    [0, "Inicialização"], [15, "Preparação"], [25, "Iniciando o jogo"],
    [35, "Ajustes"], [45, "Aguardando abertura"], [55, "Procurando janela"],
    [70, "Janela identificada"], [85, "Confirmando imagem"],
    [95, "Finalizando"], [100, "Concluído"],
  ];
  const stageInputs = [];
  stageEditor.append(
    el("h2", "Editor de etapas"),
    el("p", "Edite todas as mensagens da abertura. Os textos funcionam no jogo real e no teste de 10 segundos. Salve o layout para guardar as alterações.", "stageHelp"),
  );
  const messages = el("div", null, "stageMessages");
  for (let index = 0; index < stageDefinitions.length; index++) {
    const [percent, name] = stageDefinitions[index];
    const entry = el("div", null, "stageMessageRow");
    const input = inp(doc.stageTexts?.[index] ?? factory.stageTexts?.[index] ?? "");
    input.maxLength = 96;
    input.dataset.stageIndex = String(index);
    input.setAttribute("aria-label", "Mensagem da etapa " + percent + "%");
    input.onchange = () => {
      if (!doc.stageTexts) doc.stageTexts = [...factory.stageTexts];
      if (doc.stageTexts[index] === input.value) return;
      remember();
      doc.stageTexts[index] = input.value;
      refresh();
    };
    const preview = btn("Prévia", () => {
      previewPercent = percent;
      scrub.value = String(percent);
      scrubLabel.textContent = "Progresso: " + percent + "%";
      requestFrame();
    });
    preview.title = "Mostrar a mensagem da etapa " + percent + "% no loading";
    entry.append(el("span", percent + "% · " + name), input, preview);
    messages.append(entry);
    stageInputs.push(input);
  }
  stageEditor.append(messages, btn("Restaurar textos originais", () => {
    if (!confirm("Restaurar todas as mensagens das etapas para os textos originais?")) return;
    remember();
    doc.stageTexts = [...factory.stageTexts];
    refresh();
  }));
  function syncStageInputs() {
    for (let index = 0; index < stageInputs.length; index++) {
      if (document.activeElement !== stageInputs[index]) {
        stageInputs[index].value = doc.stageTexts?.[index] ?? factory.stageTexts?.[index] ?? "";
      }
    }
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
    const ids = new Set(doc.elements.map((node) => node.id));
    selected = new Set([...selected].filter((id) => ids.has(id)));
    // Empty selection is intentional; never select the first layer again
    // when the user clicks the background or disables a layer.
    if (selected.size && !selected.has(sel)) sel = [...selected][0];
    if (!selected.size) sel = "";
    outlines();
    properties();
    syncStageInputs();
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
    } else if (e.key.startsWith("Arrow") && selected.size) {
      e.preventDefault();
      const v = e.shiftKey ? 10 : 1;
      remember();
      for (const n of selectedNodes()) {
        n.x += { ArrowLeft: -v, ArrowRight: v }[e.key] || 0;
        n.y += { ArrowUp: -v, ArrowDown: v }[e.key] || 0;
        bounds(n);
      }
      refresh();
    }
  }
  main.onkeydown = keys;
  function float(time) {
    raf = 0;
    if (page !== serial || !anim) {
      lastAnimationFrame = 0;
      return;
    }
    const dt = lastAnimationFrame ? Math.max(0, Math.min(0.05, (time - lastAnimationFrame) / 1000)) : 1 / 60;
    lastAnimationFrame = time;
    const damping = 1 - Math.exp(-dt / 0.14);
    const finishing = previewPercent >= 95;
    const heightFactor = stage.clientHeight / 900;
    const bob = 11 * Math.sin((time / 2900) * 2 * Math.PI) * heightFactor;
    const elapsed = Math.max(0, time - finalPreviewAt);
    const u = Math.min(1, elapsed / 460);
    const pulse = previewPercent === 100 && finalPreviewAt ? Math.sin(Math.PI * u) ** 2 : 0;
    const desiredY = (finishing ? 0 : bob) - 9 * pulse * heightFactor;
    const desiredScale = (finishing ? 1.035 : 1) + 0.065 * pulse;
    motionY += (desiredY - motionY) * damping;
    motionScale += (desiredScale - motionScale) * damping;
    if (floating) {
      floating.style.transform = "translateY(" + motionY.toFixed(3) + "px) scale(" + motionScale.toFixed(5) + ")";
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
  let testBusy = false;
  const testButton = btn("Testar loading (10 s)", async () => {
    if (testBusy) return;
    testBusy = true;
    testButton.disabled = true;
    testButton.textContent = "Abrindo teste de loading…";
    const labelTimer = setTimeout(() => {
      if (testBusy) testButton.textContent = "Teste aberto — Esc para sair";
    }, 950);
    try {
      // Test unsaved layout coordinates without writing to the game files.
      await api.TestLoading(g.ID, structuredClone(doc));
      note("Teste de loading encerrado");
    } finally {
      clearTimeout(labelTimer);
      testBusy = false;
      testButton.disabled = false;
      testButton.textContent = "Testar loading (10 s)";
    }
  });
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
        if (confirm("Restaurar os sete elementos de fábrica?")) {
          remember();
          doc = structuredClone(factory);
          refresh();
        }
      }),
      btn("Print do layout salvo (50%)", () => api.Run(g.ID, "--preview")),
      testButton,
    ),
  );
  refresh();
}
