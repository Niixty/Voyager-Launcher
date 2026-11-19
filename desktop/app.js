"use strict";
const api = window.go.main.Studio,
  main = document.querySelector("main");
const $ = (id) => document.getElementById(id);
let games = [],
  selected,
  query = "",
  list = false,
  sort = "title",
  timer,
  page = 0,
  dirty = false;
function el(tag, txt, cls) {
  const e = document.createElement(tag);
  if (txt != null) e.textContent = txt;
  if (cls) e.className = cls;
  return e;
}
function add(host, ...nodes) {
  host.append(...nodes);
  return host;
}
function note(msg) {
  const n = $("notice");
  n.textContent = msg;
  n.hidden = false;
  clearTimeout(timer);
  timer = setTimeout(() => (n.hidden = true), 5200);
}
async function safe(fn) {
  try {
    await fn();
  } catch (e) {
    note(String(e));
  }
}
function btn(text, fn, cls) {
  const b = el("button", text, cls);
  b.type = "button";
  b.onclick = () => safe(fn);
  return b;
}
function row(...nodes) {
  return add(el("div", null, "row"), ...nodes);
}
function field(name, input) {
  return add(el("label", null, "field"), el("span", name), input);
}
function inp(v = "", type = "text") {
  const i = el("input");
  i.type = type;
  i.value = v ?? "";
  return i;
}
function options(pairs, v) {
  const s = el("select");
  for (const [k, n] of pairs) {
    const o = el("option", n);
    o.value = k;
    s.append(o);
  }
  s.value = v;
  return s;
}
function clear(which) {
  page++;
  main.replaceChildren();
  for (const id of ["lib", "prefs"])
    $(id).classList.toggle("active", id === which);
}
function leave() {
  if (dirty && !confirm("Descartar alterações sem salvar?")) return false;
  dirty = false;
  return true;
}
function header(title, ...actions) {
  main.append(add(el("header"), el("h1", title), row(...actions)));
}
function image(id, kind) {
  const i = el("img");
  i.alt = "";
  i.loading = "lazy";
  i.src = "/art/" + encodeURIComponent(id) + "/" + kind + "?v=" + stamp;
  i.onerror = () => i.remove();
  return i;
}
let stamp = Date.now();
async function library() {
  clear("lib");
  const token = page;
  games = await api.Library();
  if (token !== page) return;
  header(
    "Biblioteca",
    btn("Importar", async () => {
      note((await api.Import()) + " jogo(s) importado(s)");
      await library();
    }),
    btn("+ Adicionar jogo", () => form(), "primary"),
  );
  const search = inp(query, "search");
  search.placeholder = "Buscar por jogo ou plataforma";
  search.setAttribute("aria-label", "Buscar jogos");
  const order = options(
    [
      ["title", "Nome"],
      ["platform", "Plataforma"],
    ],
    sort,
  );
  const view = btn(list ? "Grade" : "Lista", () => {
    list = !list;
    view.textContent = list ? "Grade" : "Lista";
    paint();
  });
  main.append(add(el("div", null, "row toolbar"), search, order, view));
  const cards = el("div");
  main.append(cards);
  function paint() {
    cards.className = list ? "list" : "cards";
    cards.replaceChildren();
    const rows = games
      .filter((g) =>
        [g.ID, g.Config.title, g.Config.platform]
          .join(" ")
          .toLocaleLowerCase()
          .includes(query.toLocaleLowerCase()),
      )
      .sort((a, b) =>
        (a.Config[sort] || "").localeCompare(b.Config[sort] || "", "pt-BR"),
      );
    for (const g of rows) {
      const b = btn(
        "",
        () => (g.Problem ? note(g.Problem) : detail(g)),
        "card",
      );
      const cover = el("div", null, "cover");
      const i = image(g.ID, "grid");
      i.onerror = () =>
        cover.replaceChildren(el("span", (g.Config.title || g.ID).slice(0, 1)));
      cover.append(i);
      b.append(
        cover,
        add(
          el("div", null, "meta"),
          el("strong", g.Config.title || g.ID),
          el("small", g.Problem ? "Configuração inválida" : g.Config.platform),
        ),
      );
      cards.append(b);
    }
    if (!rows.length)
      cards.append(
        el(
          "p",
          query
            ? "Nenhum jogo encontrado."
            : "Adicione seu primeiro jogo ou importe uma pasta Games.",
          "muted",
        ),
      );
  }
  search.oninput = () => {
    query = search.value;
    paint();
  };
  order.onchange = () => {
    sort = order.value;
    paint();
  };
  paint();
}
function detail(g, section = "Resumo") {
  selected = g;
  clear("lib");
  header(
    g.Config.title,
    btn("Testar jogo", () => api.Run(g.ID, "--game"), "primary"),
  );
  const tabs = row();
  tabs.classList.add("tabs");
  for (const name of ["Resumo", "Imagens", "Editor de loading", "Sunshine"]) {
    const b = btn(name, () => {
      if (leave()) detail(g, name);
    });
    b.setAttribute("aria-pressed", name === section);
    tabs.append(b);
  }
  main.append(tabs);
  if (section === "Editor de loading") {
    safe(() => editor(g));
    return;
  }
  if (section === "Imagens") {
    const area = el("div", null, "art");
    main.append(area);
    for (const [kind, title, size] of [
      ["hero", "Hero", "1920 × 620"],
      ["grid", "Grid", "600 × 900"],
      ["cartridge", "Cartridge", "1536 × 1024"],
    ]) {
      const holder = el("figure", null, "panel"),
        i = image(g.ID, kind);
      i.onerror = () => i.replaceWith(el("p", "Sem imagem", "muted"));
      holder.append(
        el("h3", title),
        el("figcaption", size),
        i,
        btn("Selecionar " + title, async () => {
          const choice = await api.PickArt();
          if (!choice) return;
          const dialog = el("dialog"),
            preview = el("img");
          preview.src = choice.preview;
          preview.alt = "Prévia " + title;
          const dismiss = () => dialog.close();
          dialog.onclose = () => dialog.remove();
          dialog.append(
            el("h2", title),
            preview,
            row(
              btn("Cancelar", dismiss),
              btn(
                "Salvar imagem",
                async () => {
                  await api.SetArt(g.ID, kind, choice.path);
                  dismiss();
                  stamp++;
                  detail(g, section);
                  note("Imagem salva");
                },
                "primary",
              ),
            ),
          );
          document.body.append(dialog);
          dialog.showModal();
        }),
        btn("Remover", async () => {
          if (confirm("Mover esta imagem para o backup?")) {
            await api.RemoveArt(g.ID, kind);
            stamp++;
            detail(g, section);
          }
        }),
      );
      area.append(holder);
    }
    return;
  }
  if (section === "Sunshine") {
    const p = el("section", null, "panel");
    main.append(p);
    p.append(
      el("h2", "Comando do aplicativo"),
      el("p", "Cole este comando no campo Command do aplicativo no Sunshine."),
    );
    safe(async () => {
      const cmd = await api.Command(g.ID);
      const box = inp(cmd);
      box.readOnly = true;
      p.append(
        field("Command", box),
        btn("Copiar comando", () => window.runtime.ClipboardSetText(cmd)),
        btn("Copiar caminho da capa", async () =>
          window.runtime.ClipboardSetText(await api.CoverPath(g.ID)),
        ),
      );
    });
    return;
  }
  const hero = el("div", null, "hero"),
    im = image(g.ID, "hero");
  im.onerror = () => hero.remove();
  hero.append(im);
  main.append(
    hero,
    el("small", g.Config.platform),
    el("p", g.Config.description || ""),
    row(
      btn("Editar jogo", () => form(g)),
      btn("Duplicar", async () => {
        const id = prompt("ID da cópia", g.ID + "-copia");
        if (id) {
          await api.CopyGame(g.ID, id);
          await library();
        }
      }),
      btn("Arquivar", async () => {
        if (
          confirm(
            "Arquivar " +
              g.Config.title +
              "? Os arquivos ficam em Games-archive.",
          )
        ) {
          await api.Archive(g.ID);
          await library();
        }
      }),
    ),
  );
}
function form(g) {
  clear("lib");
  const c = {
    ...(g?.Config || { mode: "retroarch", platform: "", title: "" }),
  };
  header(g ? "Editar jogo" : "Adicionar jogo");
  const f = el("form", null, "panel"),
    fields = el("div", null, "fields");
  main.append(f);
  f.append(fields);
  const id = inp(g?.ID);
  id.required = true;
  id.pattern = "[A-Za-z0-9_-]{1,70}";
  id.disabled = !!g;
  fields.append(field("ID usado pelo Sunshine", id));
  function control(k, title, host, kind) {
    const i = k === "description" ? el("textarea") : inp();
    i.value = c[k] || "";
    i.oninput = () => {
      c[k] = i.value;
      dirty = true;
    };
    if (k === "title") i.required = true;
    const wrap = kind
      ? row(
          i,
          btn("Procurar", async () => {
            const p = await api.Pick(kind);
            if (p) {
              i.value = p;
              i.oninput();
            }
          }),
        )
      : i;
    host.append(field(title, wrap));
  }
  control("title", "Nome", fields);
  control("platform", "Plataforma", fields);
  control("coreName", "Sistema / núcleo", fields);
  control("description", "Descrição", fields);
  const mode = options(
    [
      ["retroarch", "RetroArch"],
      ["executavel", "Executável ou atalho Windows"],
    ],
    c.mode === "retroarch" ? "retroarch" : "executavel",
  );
  fields.append(field("Tipo de execução", mode));
  const launch = el("div", null, "fields panel");
  f.append(launch);
  function execution() {
    launch.replaceChildren();
    if (mode.value === "retroarch") {
      control("retroarchPath", "RetroArch", launch, "exe");
      control("corePath", "Núcleo", launch, "core");
      control("romPath", "ROM", launch, "rom");
    } else {
      c.executablePath = c.executablePath || c.shortcutPath;
      control("executablePath", "Executável ou atalho", launch, "exe");
      control("arguments", "Argumentos", launch);
      control("workingDirectory", "Pasta de trabalho", launch);
    }
    control("waitForProcess", "Processo esperado (opcional)", launch);
  }
  mode.onchange = () => {
    dirty = true;
    execution();
  };
  execution();
  f.oninput = () => (dirty = true);
  const save = el("button", "Salvar jogo", "primary");
  save.type = "submit";
  f.append(
    row(
      save,
      btn("Cancelar", () => {
        if (leave()) return g ? detail(g) : library();
      }),
    ),
  );
  f.onsubmit = (e) => {
    e.preventDefault();
    safe(async () => {
      c.mode =
        mode.value === "retroarch"
          ? "retroarch"
          : /\.lnk$/i.test(c.executablePath)
            ? "atalho"
            : "executavel";
      await api.SaveGame(id.value, c, !g);
      dirty = false;
      games = await api.Library();
      detail(games.find((x) => x.ID === id.value));
      note("Jogo salvo");
    });
  };
}
const themes = {
  cinza: "#b6bdc8",
  roxo: "#ae80ff",
  verde: "#6bdc99",
  vermelho: "#ff767c",
  azul: "#67adff",
  laranja: "#ffac60",
  rosa: "#ff8ac9",
  ciano: "#64dedf",
};
async function prefs() {
  clear("prefs");
  const t = page,
    c = await api.Prefs();
  if (t !== page) return;
  header("Configurações");
  const f = el("div", null, "panel");
  main.append(f);
  f.append(el("h2", "Aparência e loading"));
  const fields = el("div", null, "fields");
  f.append(fields);
  const theme = options(
    Object.keys(themes).map((k) => [k, k]),
    c.theme,
  );
  fields.append(field("Cor de destaque", theme));
  theme.onchange = () => {
    c.theme = theme.value;
    dirty = true;
  };
  for (const [k, name, min, max] of [
    ["postOpenSeconds", "Segundos após detectar a janela", 1, 15],
    ["backgroundDim", "Escurecimento (%)", 30, 95],
  ]) {
    const i = inp(c[k], "number");
    i.min = min;
    i.max = max;
    fields.append(field(name, i));
    i.onchange = () => {
      c[k] = Math.min(max, Math.max(min, Number(i.value)));
      i.value = c[k];
      dirty = true;
    };
  }
  for (const [k, n] of [
    ["showDescription", "Descrição"],
    ["showPercentage", "Porcentagem"],
    ["showPhase", "Etapa atual"],
    ["blurBackground", "Desfoque"],
    ["animateCartridge", "Animar cartucho"],
  ]) {
    const i = inp("", "checkbox");
    i.checked = c[k];
    i.onchange = () => {
      c[k] = i.checked;
      dirty = true;
    };
    f.append(add(el("label", null, "row"), i, el("span", n)));
  }
  f.append(
    btn(
      "Salvar preferências",
      async () => {
        await api.SavePrefs(c);
        dirty = false;
        document.documentElement.style.setProperty("--accent", themes[c.theme]);
        note("Preferências salvas");
      },
      "primary",
    ),
  );
  const d = el("section", null, "panel");
  main.append(d);
  d.append(
    el("h2", "Atualizações e diagnóstico"),
    el("p", $("version").textContent),
    btn("Verificar atualizações", async () => {
      note("Consultando GitHub…");
      if ((await api.Version()).endsWith("-preview")) {
        note("Versão de teste. Atualizações automáticas estão pausadas.");
        return;
      }
      const v = await api.CheckUpdate();
      note(
        v
          ? "Versão " + v + " pronta para instalar ao fechar."
          : "Você está na versão atual.",
      );
    }),
    btn("Verificar biblioteca", async () => {
      const lines = await api.Diagnose();
      d.append(
        el("pre", lines.join("\n") || "Arquivos de execução encontrados."),
      );
    }),
  );
}
$("lib").onclick = () => {
  if (leave()) safe(library);
};
$("prefs").onclick = () => {
  if (leave()) safe(prefs);
};
safe(async () => {
  const c = await api.Prefs();
  document.documentElement.style.setProperty("--accent", themes[c.theme]);
  $("version").textContent = "Voyager " + (await api.Version());
  await library();
});
