// Exercise the production WebView2 bridge in an isolated installation on Windows.
import { spawn, execFileSync } from "node:child_process";
import { once } from "node:events";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import assert from "node:assert/strict";

const dir = await fs.mkdtemp(path.join(os.tmpdir(), "voyager-ui-"));
const exe = path.join(dir, "VoyagerLauncher.exe");
const pause = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
await fs.copyFile("dist/VoyagerLauncher.exe", exe);
const p = spawn(exe, [], {
  cwd: dir,
  env: {
    ...process.env,
    WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS: "--remote-debugging-port=19327",
    WEBVIEW2_USER_DATA_FOLDER: path.join(dir, "profile"),
  },
});
let socket;
try {
  let page,
    targets = [];
  for (let i = 0; i < 100 && !page; i++) {
    if (p.exitCode !== null)
      throw Error("O gerenciador encerrou: " + p.exitCode);
    try {
      targets = await (await fetch("http://127.0.0.1:19327/json/list")).json();
      page = targets.find((t) => t.type === "page" && t.url.includes("wails"));
    } catch {}
    if (!page) await pause(300);
  }
  if (!page) {
    console.error(
      "Páginas WebView",
      targets.map((t) => [t.type, t.url]),
    );
    console.error(
      execFileSync(
        "powershell.exe",
        [
          "-NoProfile",
          "-Command",
          "Get-Process VoyagerLauncher,msedgewebview2 -ErrorAction SilentlyContinue | Select-Object Name,Id,MainWindowTitle | Out-String; exit 0",
        ],
        { encoding: "utf8" },
      ),
    );
    const log = path.join(
      process.env.LOCALAPPDATA,
      "VoyagerLauncher",
      "update-check.log",
    );
    console.error(
      await fs.readFile(log, "utf8").catch(() => "Sem log de inicialização"),
    );
  }
  assert.ok(page, "WebView2 do Voyager não abriu");
  socket = new WebSocket(page.webSocketDebuggerUrl);
  await once(socket, "open");
  let id = 0;
  const replies = new Map();
  socket.onmessage = ({ data }) => {
    const m = JSON.parse(data);
    replies.get(m.id)?.(m);
    replies.delete(m.id);
  };
  const send = (method, params) =>
    new Promise((resolve, reject) => {
      const n = ++id;
      const timer = setTimeout(
        () => reject(Error("Tempo esgotado: " + method)),
        15000,
      );
      replies.set(n, (m) => {
        clearTimeout(timer);
        m.error ? reject(Error(m.error.message)) : resolve(m.result);
      });
      socket.send(JSON.stringify({ id: n, method, params }));
    });
  const js = async (expression) => {
    const r = await send("Runtime.evaluate", {
      expression,
      awaitPromise: true,
      returnByValue: true,
    });
    if (r.exceptionDetails)
      throw Error(
        r.exceptionDetails.exception?.description || r.exceptionDetails.text,
      );
    return r.result.value;
  };
  const waitFor = async (expression) => {
    for (let i = 0; i < 80; i++) {
      if (await js(expression)) return;
      await pause(150);
    }
    throw Error(
      "Estado ausente: " +
        expression +
        "\n" +
        (await js("document.body.innerText")),
    );
  };
  await waitFor("document.querySelector('h1')?.textContent==='Biblioteca'");
  const click = async (label) =>
    js(
      `[...document.querySelectorAll('button')].find(b=>b.textContent===${JSON.stringify(label)}).click()`,
    );
  await click("+ Adicionar jogo");
  await waitFor("document.querySelector('form')");
  await js(
    `(()=>{const inputs=document.querySelectorAll('form input');const set=(i,v)=>{i.value=v;i.dispatchEvent(new Event('input',{bubbles:true}))};set(inputs[0],'Smoke');set(inputs[1],'Órbita');set(inputs[2],'Windows');const mode=document.querySelector('form select');mode.value='executavel';mode.dispatchEvent(new Event('change'));set(document.querySelector('.fields.panel input'),'C:\\\\Windows\\\\notepad.exe');document.querySelector('form').requestSubmit()})()`,
  );
  await waitFor("document.querySelector('h1')?.textContent==='Órbita'");
  await click("Editor de loading");
  await waitFor(
    "document.querySelector('.stage')?.style.backgroundImage.includes('data:image/png')",
  );
  await click("Salvar layout");
  await waitFor(
    "document.querySelector('#notice').textContent==='Layout salvo'",
  );
  assert.equal(
    JSON.parse(
      await fs.readFile(
        path.join(dir, "Games", "Smoke", "loading_layout.json"),
      ),
    ).elements.length,
    8,
  );
  await click("Configurações");
  await waitFor("document.querySelector('h1')?.textContent==='Configurações'");
  await click("Salvar preferências");
  await waitFor(
    "document.querySelector('#notice').textContent==='Preferências salvas'",
  );
  await click("Biblioteca");
  await waitFor(
    "document.querySelector('.card strong')?.textContent==='Órbita'",
  );
  await fs.mkdir("evidence", { recursive: true });
  const shot = await send("Page.captureScreenshot", { format: "png" });
  await fs.writeFile(
    "evidence/biblioteca.png",
    Buffer.from(shot.data, "base64"),
  );
  console.log(
    "WebView2: biblioteca, cadastro, layout GDI+, persistência e navegação OK.",
  );
} finally {
  socket?.close();
  p.kill();
  await pause(500);
  await fs
    .rm(dir, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 })
    .catch(() => {});
}
