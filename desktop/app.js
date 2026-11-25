"use strict";
const api = window.go.main.Studio,
  main = document.querySelector("main");
const $ = (id) => document.getElementById(id);
let games = [],
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
    btn("Big Picture", openBigPicture),
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
      b.oncontextmenu = (ev) => {
        ev.preventDefault();
        if (g.Problem) return;
        document.querySelector(".game-menu")?.remove();
        const menu = el("div", null, "game-menu");
        menu.setAttribute("popover", "auto");
        for (const [text, fn] of [
          ["Testar jogo", () => api.Run(g.ID, "--game")],
          ["Editar jogo", () => form(g)],
          ["Editor de loading", () => detail(g, "Editor de loading")],
        ])
          menu.append(
            btn(text, () => {
              menu.hidePopover();
              return fn();
            }),
          );
        menu.ontoggle = () => {
          if (!menu.matches(":popover-open")) menu.remove();
        };
        document.body.append(menu);
        menu.showPopover();
        menu.style.left =
          Math.max(0, Math.min(ev.clientX, innerWidth - menu.offsetWidth - 8)) +
          "px";
        menu.style.top =
          Math.max(
            0,
            Math.min(ev.clientY, innerHeight - menu.offsetHeight - 8),
          ) + "px";
        menu.querySelector("button").focus();
      };
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
  clear("lib");
  const play = btn("Testar jogo", () => api.Run(g.ID, "--game"), "primary");
  if (section === "Resumo") {
    const banner = el("section", null, "detail-banner"),
      backdrop = image(g.ID, "hero");
    backdrop.className = "backdrop";
    banner.append(backdrop);
    const info = el("div", null, "banner-info");
    info.append(el("small", g.Config.platform), el("h1", g.Config.title), play);
    banner.append(info);
    main.append(banner);
  } else {
    header(g.Config.title, play);
  }
  const tabs = row();
  tabs.classList.add("tabs");
  for (const name of ["Resumo", "Imagens", "Editor de loading", "VibePollo"]) {
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
      if (kind !== "cartridge") {
        holder.append(btn("Procurar na base de dados", () => steamGridBrowser(g, kind)));
      } else {
        holder.append(el("small", "Cartridge: seleção manual por enquanto."));
      }
      area.append(holder);
    }
    return;
  }
  if (section === "VibePollo") {
    const p = el("section", null, "panel");
    main.append(p);
    p.append(
      el("h2", "Adicionar aplicativo no VibePollo"),
      el("p", "Use este perfil no painel do VibePollo → Aplicativos → Adicionar aplicativo. Serve para jogos EXE, atalhos .lnk e RetroArch: o Voyager escolhe a execução configurada."),
    );
    safe(async () => {
      const cmd = await api.Command(g.ID);
      const cover = await api.CoverPath(g.ID);
      const workingDir = cmd.match(/^"([^"]+)"/)?.[1]?.replace(/[\\/][^\\/]+$/, "") || "";
      const values = [
        ["Application Name / Nome", g.Config.title || "", "Nome exibido no Moonlight"],
        ["Image / Imagem (PNG)", cover || "", "Use a capa Grid do jogo. Se estiver vazia, selecione uma Grid primeiro."],
        ["Command / Comando principal", "", "Deixe vazio: a tela de loading termina antes do jogo."],
        ["Detached Commands / Comandos em segundo plano", cmd, "Adicione como um comando detached para não interromper o streaming ao fechar o loading."],
        ["Working Directory / Pasta de trabalho", workingDir, "Pasta do Voyager Launcher; pode deixar em branco se o VibePollo reconhecer o caminho do executável."],
      ];
      for (const [title, value, explanation] of values) {
        const box = inp(value);
        box.readOnly = true;
        p.append(field(title, box), el("small", explanation),
          btn("Copiar " + title.split(" / ")[0], () => window.runtime.ClipboardSetText(value)));
      }
      p.append(
        el("h3", "Outras opções do VibePollo"),
        el("p", "Continue streaming if the application exits quickly: não é necessário usando Detached Commands. Continue streaming until all app processes exit: deixe desmarcado inicialmente. Run as Admin, Display/Resolution, Global Prep e demais overrides: mantenha seus valores atuais, salvo necessidade específica."),
        btn("Copiar instruções completas", () => window.runtime.ClipboardSetText(
          values.map(([key,value]) => key + ": " + (value || "(vazio)")).join("\n")
        )),
        btn("Abrir painel local do VibePollo", () => window.runtime.BrowserOpenURL("https://127.0.0.1:47990")),
      );
    });
    return;
  }
  main.append(
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
  fields.append(field("ID do jogo no Voyager (usado pelo VibePollo)", id));
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
  const font = options(
    ["Segoe UI", "Arial", "Tahoma", "Verdana"].map((n) => [n, n]),
    c.interfaceFont || "Segoe UI",
  );
  fields.append(field("Tipografia da interface", font));
  font.onchange = () => {
    c.interfaceFont = font.value;
    dirty = true;
  };
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
        document.body.style.fontFamily = c.interfaceFont || "Segoe UI";
        note("Preferências salvas");
      },
      "primary",
    ),
  );
  const sgdb = el("section", null, "panel");
  main.append(sgdb);
  sgdb.append(
    el("h2", "Integração SteamGridDB"),
    el("p", "Gere uma chave gratuita em steamgriddb.com/profile/preferences/api. A chave é protegida no Windows e nunca é incluída nos arquivos do jogo ou no GitHub."),
  );
  const apiLink = el("a", "Obter minha chave da API");
  apiLink.href = "https://www.steamgriddb.com/profile/preferences/api";
  apiLink.target = "_blank";
  apiLink.rel = "noopener noreferrer";
  sgdb.append(apiLink);
  const keyField = inp("", "password");
  keyField.autocomplete = "off";
  keyField.placeholder = "Cole sua chave da API aqui";
  const keyStatus = el("p", "Verificando chave…");
  sgdb.append(field("Chave SteamGridDB (não será exibida novamente)", keyField), keyStatus);
  safe(async () => {
    keyStatus.textContent = (await api.SteamGridKeyConfigured())
      ? "Chave configurada neste Windows."
      : "Chave ainda não configurada.";
  });
  sgdb.append(row(
    btn("Salvar chave", async () => {
      if (!keyField.value.trim()) throw Error("Informe a chave antes de salvar.");
      await api.SaveSteamGridKey(keyField.value.trim());
      keyField.value = "";
      keyStatus.textContent = "Chave configurada neste Windows.";
      note("Chave SteamGridDB salva");
    }, "primary"),
    btn("Remover chave", async () => {
      if (!confirm("Remover a chave SteamGridDB salva neste computador?")) return;
      await api.RemoveSteamGridKey();
      keyField.value = "";
      keyStatus.textContent = "Chave ainda não configurada.";
      note("Chave SteamGridDB removida");
    }),
  ));

  // Version verification never downloads or installs anything. Only the
  // explicitly confirmed "Aplicar atualização" path stages and restarts.
  const d = el("section", null, "panel update-panel");
  main.append(d);
  const head = el("div", null, "update-heading");
  head.append(
    add(el("div"), el("h2", "Atualizações e diagnóstico"),
      el("p", "Verifique novas versões, instale com segurança e consulte o histórico do aplicativo.")),
  );
  const status = el("div", "Consultando versão instalada…", "update-status");
  head.append(status);
  const versions = el("div", null, "update-versions");
  const installed = add(el("div", null, "update-version"),
    el("small", "VERSÃO INSTALADA"), el("strong", "—"));
  const latest = add(el("div", null, "update-version"),
    el("small", "VERSÃO MAIS RECENTE"), el("strong", "Verificando…"));
  versions.append(installed, latest);
  const actions = el("div", null, "update-actions");
  const checkButton = btn("Verificar atualizações", checkForUpdates);
  const applyButton = btn("Aplicar atualização", applyUpdate, "primary update-apply");
  applyButton.hidden = true;
  const historyButton = btn("Histórico", async () => {
    const text = await api.UpdateHistory();
    showUpdateDetails("Histórico de atualizações", text);
  });
  const diagnosticsButton = btn("Verificar biblioteca", async () => {
    const lines = await api.Diagnose();
    showUpdateDetails("Diagnóstico da biblioteca", lines.join("\n") || "Nenhum problema encontrado.");
  });
  actions.append(checkButton, applyButton, historyButton, diagnosticsButton);
  const details = el("div", null, "update-details");
  details.hidden = true;
  d.append(head, versions, actions, details);

  function showUpdateDetails(title, contents) {
    details.replaceChildren(el("h3", title), el("pre", contents));
    details.hidden = false;
  }
  function renderUpdateInfo(info) {
    if (page !== t || !info) return;
    installed.querySelector("strong").textContent = info.current || "—";
    latest.querySelector("strong").textContent =
      info.latest || (info.preview ? "Prévia de desenvolvimento" : "Nenhuma versão nova");
    const available = Boolean(info.available && !info.preview);
    applyButton.hidden = !available;
    applyButton.disabled = Boolean(info.downloading);
    status.classList.remove("available", "checking", "failed", "current");
    if (info.preview) {
      status.textContent = "Versão de desenvolvimento — verificação automática desativada";
      status.classList.add("current");
    } else if (info.downloading) {
      status.textContent = "Baixando atualização — " + (info.progress || 0) + "%";
      status.classList.add("checking");
    } else if (info.checking) {
      status.textContent = "Procurando versões novas no GitHub…";
      status.classList.add("checking");
    } else if (info.error) {
      status.textContent = "Não foi possível verificar: " + info.error;
      status.classList.add("failed");
    } else if (available) {
      status.textContent = "Nova versão disponível" + (info.ready ? " — download já preparado" : "");
      status.classList.add("available");
    } else {
      status.textContent = "Você está usando a versão mais recente";
      status.classList.add("current");
    }
  }
  async function checkForUpdates() {
    checkButton.disabled = true;
    status.textContent = "Verificando novas versões no GitHub…";
    status.classList.add("checking");
    try {
      renderUpdateInfo(await api.CheckUpdateStatus());
    } catch (error) {
      renderUpdateInfo(await api.UpdateInfo());
      status.textContent = "Não foi possível verificar: " + String(error);
      status.classList.add("failed");
    } finally {
      if (page === t) checkButton.disabled = false;
    }
  }
  async function applyUpdate() {
    const info = await api.UpdateInfo();
    if (!info.available || info.preview) {
      renderUpdateInfo(info);
      return;
    }
    if (dirty && !confirm("Existem preferências não salvas. Descartá-las e instalar a atualização?")) return;
    if (!confirm("Instalar a versão " + info.latest + "? O Voyager será fechado e reaberto automaticamente após a instalação.")) return;

    const dialog = el("dialog", null, "update-dialog");
    const body = el("div", null, "update-dialog-content");
    const heading = el("h2", "Atualizando Voyager");
    const message = el("p", "Preparando download seguro da versão " + info.latest + "…");
    const percent = el("strong", "0%", "update-percentage");
    const progress = el("progress");
    progress.max = 100;
    progress.value = 0;
    const help = el("small", "O arquivo será verificado por SHA-256 antes da instalação.");
    body.append(heading, message, percent, progress, help);
    dialog.append(body);
    document.body.append(dialog);
    dialog.showModal();
    // Prevent the progress dialog from being closed midway through an EXE download.
    dialog.addEventListener("cancel", (event) => event.preventDefault());
    let finished = false;
    const paintProgress = (state) => {
      if (!dialog.open || !state) return;
      const current = Math.max(0, Math.min(100, Number(state.progress) || 0));
      progress.value = current;
      percent.textContent = current + "%";
      message.textContent = state.stage || "Preparando atualização…";
    };
    const interval = setInterval(() => {
      if (!finished) api.UpdateInfo().then(paintProgress).catch(() => {});
    }, 180);
    try {
      await api.PrepareUpdate();
      finished = true;
      clearInterval(interval);
      paintProgress(await api.UpdateInfo());
      message.textContent = "Download concluído. Fechando para instalar e reabrir o Voyager…";
      help.textContent = "Se o Windows impedir a substituição, a versão anterior será preservada.";
      await api.InstallUpdateNow();
      // OnShutdown launches VoyagerUpdater.exe, which waits for the manager
      // process to exit, replaces the validated EXE and starts it again.
      window.runtime.Quit();
    } catch (error) {
      finished = true;
      clearInterval(interval);
      heading.textContent = "Atualização não concluída";
      message.textContent = String(error);
      help.textContent = "O Voyager atual continuará funcionando normalmente.";
      body.append(btn("Fechar", () => dialog.close(), "primary"));
      dialog.addEventListener("close", () => dialog.remove(), { once: true });
      renderUpdateInfo(await api.UpdateInfo());
    }
  }

  // Opening Settings performs another independent check. The backend also
  // probes once at launch, but never stages an EXE without user consent.
  safe(checkForUpdates);
}
// Browser de artes SteamGridDB (busca pelo nome já salvo no jogo).
async function steamGridBrowser(g, kind) {
  const dialog = el("dialog", null, "sgdb-dialog");
  const shell = el("div", null, "sgdb-shell");
  dialog.append(shell);
  dialog.onclose = () => dialog.remove();
  document.body.append(dialog);
  dialog.showModal();
  const token = {cancelled:false};
  dialog.addEventListener("close", () => { token.cancelled = true; generation++; });
  const title = el("h2", "SteamGridDB · " + (kind === "hero" ? "Heroes" : "Grids"));
  const info = el("p", "Buscando com o nome salvo do jogo: " + (g.Config.title || "(não definido)"));
  const status = el("p", "", "muted");
  const gallery = el("div", null, "sgdb-gallery");
  const gamesBox = el("div", null, "sgdb-game-picker");
  const controls = row(btn("Fechar", () => dialog.close()));
  const more = btn("Carregar mais imagens", () => safe(loadImages));
  more.hidden = true;
  controls.append(more);
  shell.append(title, info, gamesBox, status, gallery, controls);
  // Each selection has its own request generation: an old request cannot
  // append images or change the load-more button after the selected game
  // changes. The new selection can load while the old request is pending.
  let gameID = 0, currentPage = 0, generation = 0, loadingGeneration = -1;
  const showError = (error) => {
    status.textContent = String(error);
    if (/chave|API|401|403/i.test(String(error))) {
      const setup = btn("Configurar chave da API", () => {
        dialog.close();
        safe(prefs);
      });
      gamesBox.append(setup);
    }
    if (/nome do jogo/i.test(String(error))) {
      gamesBox.append(btn("Definir nome do jogo", () => { dialog.close(); form(g); }));
    }
  };
  async function loadImages() {
    if (!gameID || token.cancelled || loadingGeneration === generation) return;
    const requestGeneration = generation;
    const selectedGameID = gameID;
    const requestedPage = currentPage;
    loadingGeneration = requestGeneration;
    more.disabled = true;
    status.textContent = "Buscando imagens (página " + (requestedPage + 1) + ")…";
    try {
      const result = await api.SteamGridImages(selectedGameID, kind, requestedPage);
      if (token.cancelled || requestGeneration !== generation || gameID !== selectedGameID) return;
      for (const artwork of result.items || []) {
        const item = el("button", null, "sgdb-item");
        item.type = "button";
        const preview = el("img");
        preview.src = artwork.thumb || artwork.url;
        preview.alt = (kind === "hero" ? "Hero" : "Grid") + " " + artwork.width + " × " + artwork.height;
        preview.loading = "lazy";
        const meta = el("small", artwork.width + " × " + artwork.height);
        item.append(preview, meta);
        item.onclick = () => safe(async () => {
          if (!confirm("Usar esta imagem como " + (kind === "hero" ? "Hero" : "Grid") + " de " + g.Config.title + "?")) return;
          status.textContent = "Baixando e aplicando imagem…";
          item.disabled = true;
          try {
            await api.ApplySteamGridImage(g.ID, kind, artwork.url);
            stamp++;
            dialog.close();
            detail(g, "Imagens");
            note("Imagem SteamGridDB aplicada com sucesso");
          } finally { item.disabled = false; }
        });
        gallery.append(item);
      }
      const count = result.items?.length || 0;
      const limit = result.limit || 50;
      currentPage++;
      more.hidden = count === 0 || (result.total > 0 ? currentPage * limit >= result.total : count < limit);
      status.textContent = gallery.childElementCount
        ? gallery.childElementCount + " imagens carregadas. Escolha a que preferir."
        : "Nenhuma imagem encontrada para este jogo.";
    } catch (e) {
      if (!token.cancelled && requestGeneration === generation) showError(e);
    } finally {
      if (loadingGeneration === requestGeneration) {
        loadingGeneration = -1;
        more.disabled = false;
      }
    }
  }
  try {
    const savedTitle = await api.SteamGridSavedGameTitle(g.ID);
    info.textContent = "Buscando com o nome salvo do jogo: " + savedTitle;
    if (!(await api.SteamGridKeyConfigured())) {
      throw Error("Configure sua chave SteamGridDB em Configurações antes de pesquisar.");
    }
    const gamesFound = await api.SteamGridGames(g.ID);
    if (token.cancelled) return;
    if (!gamesFound?.length) {
      status.textContent = "Nenhum jogo encontrado na SteamGridDB para este nome.";
      return;
    }
    const chooser = options(gamesFound.map((x) => [String(x.id), x.name]), String(gamesFound[0].id));
    const normalized = (s) => String(s || "").trim().toLocaleLowerCase();
    const exact = gamesFound.find((x) => normalized(x.name) === normalized(g.Config.title));
    if (exact) chooser.value = String(exact.id);
    chooser.onchange = () => {
      generation++; // retire outstanding results from the previous game
      gameID = Number(chooser.value);
      currentPage = 0;
      gallery.replaceChildren();
      more.hidden = true;
      more.disabled = false;
      status.textContent = "Carregando imagens do jogo selecionado…";
      safe(loadImages);
    };
    gamesBox.append(field("Jogo encontrado na SteamGridDB", chooser));
    gameID = Number(chooser.value);
    await loadImages();
  } catch (e) {
    if (!token.cancelled) showError(e);
  }
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
  document.body.style.fontFamily = c.interfaceFont || "Segoe UI";
  $("version").textContent = "Voyager " + (await api.Version());
  await library();
});
