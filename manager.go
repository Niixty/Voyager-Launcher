//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

type clickable struct {
	key        string
	x, y, w, h int
}

func (b clickable) has(x, y int) bool { return x >= b.x && y >= b.y && x < b.x+b.w && y < b.y+b.h }

type managerState struct {
	hwnd               uintptr
	tab                int
	activeID           string
	creating           bool
	ids                []string
	scroll             int
	status             string
	inputs             map[string]uintptr
	labels             map[string]clickable
	buttons            []clickable
	heroPath, gridPath, cartridgePath string
	hero, grid, cartridge         *img
	darkBrush          uintptr
	editFont           uintptr
	windowW, windowH   int
	mode               string
}

var manager *managerState
var managerCallback = syscall.NewCallback(managerWndProc)
var managerClassName = syscall.StringToUTF16Ptr("VoyagerManagerNativeV8")
var editKeys = []string{"ID", "Title", "Platform", "CoreName", "Description", "Exe", "Core", "ROM", "Process", "Command", "Cover"}

func managerWndProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	m := manager
	switch msg {
	case WM_CREATE:
		return 0
	case WM_GETMINMAXINFO:
		mm := (*minMaxInfo)(unsafe.Pointer(lp))
		mm.MinTrackSize = point{1040, 800}
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		var ps paintStruct
		hdc := call(pBeginPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
		if m != nil {
			r := client(hwnd)
			m.windowW = int(r.w())
			m.windowH = int(r.h())
			m.paintBuffered(hdc)
		}
		call(pEndPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_SIZE:
		if m != nil {
			m.layout()
			invalid(hwnd)
		}
		return 0
	case WM_VOYAGER_UPDATE:
		if m != nil {
			updateState.Lock()
			m.status = updateState.status
			updateState.Unlock()
			invalid(hwnd)
		}
		return 0
	case WM_CTLCOLORSTATIC, WM_CTLCOLOREDIT:
		hdc := wp
		call(pSetTextColor, hdc, 0x00f3f3f4)
		call(pSetBkColor, hdc, 0x00262222)
		return m.darkBrush
	case WM_COMMAND:
		if m != nil && (wp>>16) == 0x300 { // EN_CHANGE
			if lp == m.inputs["ID"] {
				// Atualiza apenas os campos derivados. Nao redesenhe a janela inteira.
				m.refreshCommands()
			} else if lp == m.inputs["Title"] {
				// O titulo tambem aparece no cabecalho. Redesenhe so essa area
				// e nao apague o fundo novamente a cada tecla digitada.
				m.invalidateTitle()
			}
			// Outros campos EDIT desenham seu proprio texto pelo Windows.
		}
		return 0
	case WM_LBUTTONDOWN:
		if m != nil {
			x, y := wordXY(lp)
			m.click(int(x), int(y))
		}
		return 0
	case WM_MOUSEWHEEL:
		if m != nil {
			sx, sy := wordXY(lp) // coordinates are screen-relative for WM_MOUSEWHEEL
			mouse := point{X: sx, Y: sy}
			call(pScreenToClient, hwnd, uintptr(unsafe.Pointer(&mouse)))
			bottom := m.windowH - 225
			if mouse.X >= 17 && mouse.X < 255 && mouse.Y >= 168 && int(mouse.Y) < bottom {
				rows := (m.windowH - 405) / 47
				if rows < 1 { rows = 1 }
				maxScroll := len(m.ids) - rows
				if maxScroll < 0 { maxScroll = 0 }
				oldScroll := m.scroll
				delta := int16((wp >> 16) & 0xffff)
				if delta > 0 && m.scroll > 0 { m.scroll-- }
				if delta < 0 && m.scroll < maxScroll { m.scroll++ }
				if m.scroll != oldScroll {
					region := rect{Left: 17, Top: 168, Right: 255, Bottom: int32(bottom)}
					call(pInvalidate, hwnd, uintptr(unsafe.Pointer(&region)), 0)
				}
			}
		}
		return 0
	case WM_CLOSE:
		call(pDestroyWindow, hwnd)
		return 0
	case WM_DESTROY:
		if m != nil {
			m.close()
			manager = nil
		}
		_ = launchPendingUpdate()
		call(pPostQuitMessage, 0)
		return 0
	}
	return call(pDefWindowProc, hwnd, uintptr(msg), wp, lp)
}
// invalidateTitle limita o repintar a area visivel do cabecalho.
 // Sem isso, EN_CHANGE invalidava toda a janela a cada caractere.
func (m *managerState) invalidateTitle() {
	if m.hwnd == 0 {
		return
	}
	area := rect{Left: 305, Top: 48, Right: int32(m.windowW - 270), Bottom: 115}
	if area.Right > area.Left {
		// FALSE: nao solicitar limpeza de fundo antes do WM_PAINT.
		call(pInvalidate, m.hwnd, uintptr(unsafe.Pointer(&area)), 0)
	}
}

func (m *managerState) close() {
	if m.hero != nil {
		m.hero.close()
	}
	if m.grid != nil {
		m.grid.close()
	}
	if m.cartridge != nil {
		m.cartridge.close()
	}
	if m.darkBrush != 0 {
		call(pDeleteObject, m.darkBrush)
	}
	if m.editFont != 0 {
		call(pDeleteObject, m.editFont)
	}
}
func newManager() error {
	manager = &managerState{inputs: map[string]uintptr{}, labels: map[string]clickable{}, status: "Pronto para configurar os jogos", mode: "retroarch"}
	m := manager
	m.ids = listGames()
	instance := call(pGetModuleHandle, 0)
	appIcon := call(pLoadIcon, instance, 1) // RT_GROUP_ICON principal embutido no EXE
	wc := wndClass{Size: uint32(unsafe.Sizeof(wndClass{})), Style: CS_HREDRAW | CS_VREDRAW, WndProc: managerCallback, Instance: instance, Icon: appIcon, IconSm: appIcon, Cursor: call(pLoadCursor, 0, IDC_ARROW), ClassName: uintptr(unsafe.Pointer(managerClassName))}
	if call(pRegisterClass, uintptr(unsafe.Pointer(&wc))) == 0 {
		return winError("erro ao registrar interface")
	}
	hwnd := call(pCreateWindow, 0, uintptr(unsafe.Pointer(managerClassName)), ptrStr("Voyager Launcher"), WS_OVERLAPPEDWINDOW|WS_CLIPCHILDREN, 80, 65, 1280, 810, 0, 0, instance, 0)
	if hwnd == 0 {
		return winError("erro ao criar janela")
	}
	m.hwnd = hwnd
	setDarkTitle(hwnd)
	m.darkBrush = call(pCreateSolidBrush, 0x00262222)
	negHeight := int32(-19)
	m.editFont = call(pCreateFont, uintptr(negHeight), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 0, 0, ptrStr("Segoe UI"))
	for _, k := range editKeys {
		flags := uintptr(WS_CHILD | ES_AUTOHSCROLL | WS_TABSTOP)
		if k == "Description" || k == "Command" {
			flags = WS_CHILD | ES_MULTILINE | ES_AUTOVSCROLL | WS_TABSTOP
		}
		if k == "Command" || k == "Cover" {
			flags |= ES_READONLY
		}
		handle := call(pCreateWindow, 0, ptrStr("EDIT"), ptrStr(""), flags, 0, 0, 100, 35, hwnd, 0, instance, 0)
		m.inputs[k] = handle
		call(pSendMessage, handle, WM_SETFONT, m.editFont, 1)
	}
	if len(m.ids) > 0 {
		m.load(m.ids[0])
	} else {
		m.resetNew()
	}
	m.layout()
	show(hwnd, SW_SHOW)
	call(pUpdateWindow, hwnd)
	startUpdateCheck(hwnd)
	windowsLoop()
	return nil
}
func (m *managerState) field(k string) string { return strings.TrimSpace(getText(m.inputs[k])) }
func (m *managerState) set(k, v string)       { setText(m.inputs[k], v) }
func (m *managerState) refreshCommands() {
	id := m.field("ID")
	if !validID.MatchString(id) {
		m.set("Command", "")
		m.set("Cover", "")
		return
	}
	exe, _ := os.Executable()
	m.set("Command", fmt.Sprintf("\"%s\" --game \"%s\"", exe, id))
	m.set("Cover", filepath.Join(gameDir(id), "grid.png"))
}
func (m *managerState) resetNew() {
	m.activeID = ""
	m.creating = true
	m.mode = "retroarch"
	m.heroPath = ""
	m.gridPath = ""
	m.cartridgePath = ""
	if m.cartridge != nil {
		m.cartridge.close()
		m.cartridge = nil
	}
	if m.hero != nil {
		m.hero.close()
		m.hero = nil
	}
	if m.grid != nil {
		m.grid.close()
		m.grid = nil
	}
	call(pSendMessage, m.inputs["ID"], 0x00CF, 0, 0)
	m.set("ID", "NovoJogo")
	m.set("Title", "")
	m.set("Platform", "Windows / Emulador")
	m.set("CoreName", "Jogo")
	m.set("Description", "")
	m.set("Exe", `D:\Jogos\Emuladores\RetroArch\retroarch.exe`)
	m.set("Core", "")
	m.set("ROM", "")
	m.set("Process", "")
	m.tab = 0
	m.status = "Novo jogo — preencha as informações e salve"
	m.refreshCommands()
	m.layout()
	invalid(m.hwnd)
}
func (m *managerState) load(id string) {
	c, err := readConfig(id)
	if err != nil {
		messageBox(m.hwnd, err.Error(), "Erro ao abrir jogo")
		return
	}
	m.activeID = id
	m.creating = false
	m.mode = c.Mode
	m.set("ID", id)
	call(pSendMessage, m.inputs["ID"], 0x00CF, 1, 0)
	m.set("Title", c.Title)
	m.set("Platform", c.Platform)
	m.set("CoreName", c.CoreName)
	m.set("Description", c.Description)
	if c.Mode == "executavel" || c.Mode == "atalho" {
		path := c.ExecutablePath
		if path == "" {
			path = c.ShortcutPath
		}
		m.set("Exe", path)
		m.set("Core", "")
		m.set("ROM", c.Arguments)
	} else {
		m.mode = "retroarch"
		m.set("Exe", c.RetroarchPath)
		m.set("Core", c.CorePath)
		m.set("ROM", c.RomPath)
	}
	m.set("Process", c.WaitForProcess)
	m.heroPath = artworkPath(c, id, "hero")
	m.gridPath = artworkPath(c, id, "grid")
	m.cartridgePath = artworkPath(c, id, "cartridge")
	if m.cartridge != nil { m.cartridge.close() }
	m.cartridge = loadImg(m.cartridgePath)
	if m.hero != nil {
		m.hero.close()
	}
	if m.grid != nil {
		m.grid.close()
	}
	m.hero = loadImg(m.heroPath)
	m.grid = loadImg(m.gridPath)
	m.status = "Editando: " + id
	m.refreshCommands()
	m.layout()
	invalid(m.hwnd)
}
func (m *managerState) readUI() gameConfig {
	c := gameConfig{SchemaVersion: 6, Title: m.field("Title"), Platform: m.field("Platform"), CoreName: m.field("CoreName"), Description: getText(m.inputs["Description"]), Mode: m.mode, Hero: "hero.png", Grid: "grid.png", Cartridge: "cartridge.png", WaitForProcess: m.field("Process")}
	if m.mode == "retroarch" {
		c.RetroarchPath = m.field("Exe")
		c.CorePath = m.field("Core")
		c.RomPath = m.field("ROM")
	} else {
		c.ExecutablePath = m.field("Exe")
		c.Arguments = getText(m.inputs["ROM"])
		if strings.EqualFold(filepath.Ext(c.ExecutablePath), ".lnk") {
			c.Mode = "atalho"
		} else {
			c.Mode = "executavel"
		}
	}
	return c
}
func (m *managerState) save() error {
	id := m.field("ID")
	c := m.readUI()
	if m.activeID != "" && c.Mode != "retroarch" {
		old, e := readConfig(m.activeID)
		if e == nil && strings.EqualFold(old.ExecutablePath, c.ExecutablePath) {
			c.WorkingDirectory = old.WorkingDirectory
		}
	}
	if m.activeID != "" && id != m.activeID {
		return fmt.Errorf("não é possível alterar o ID de um jogo existente. Crie outro jogo")
	}
	if err := writeConfig(id, c); err != nil {
		return err
	}
	// Keep source artwork paths untouched if saving the current built-in images.
	for _, v := range []struct {
		path, name string
		w, h       int
	}{{m.heroPath, "hero", 1920, 620}, {m.gridPath, "grid", 600, 900}} {
		if v.path != "" {
			if err := cropToPNG(v.path, filepath.Join(gameDir(id), v.name+".png"), v.w, v.h); err != nil {
				return err
			}
		}
	}
	// Preserve alpha and proportions. Cartridge does not appear in the loading yet.
	if m.cartridgePath != "" {
		if err := fitTransparentPNG(m.cartridgePath, filepath.Join(gameDir(id), "cartridge.png"), 1536, 1024); err != nil {
			return fmt.Errorf("falha ao salvar Cartridge: %w", err)
		}
	}
	m.ids = listGames()
	m.load(id)
	m.status = "Jogo salvo — configuração automática de carregamento"
	invalid(m.hwnd)
	return nil
}
func (m *managerState) selectArt(which string) {
	path := openFile(m.hwnd, "Imagens (*.png;*.jpg;*.jpeg;*.bmp)\x00*.png;*.jpg;*.jpeg;*.bmp\x00Todos (*.*)\x00*.*\x00", rootDir)
	if path == "" {
		return
	}
	switch which {
	case "hero":
		image := loadImg(path)
		if image == nil { messageBox(m.hwnd, "Não foi possível abrir essa imagem.", "Hero"); return }
		if m.hero != nil { m.hero.close() }
		m.heroPath, m.hero = path, image
	case "grid":
		image := loadImg(path)
		if image == nil { messageBox(m.hwnd, "Não foi possível abrir essa imagem.", "Grid"); return }
		if m.grid != nil { m.grid.close() }
		m.gridPath, m.grid = path, image
	case "cartridge":
		image := loadImg(path)
		if image == nil { messageBox(m.hwnd, "Não foi possível abrir essa imagem.", "Cartridge"); return }
		if m.cartridge != nil { m.cartridge.close() }
		m.cartridgePath, m.cartridge = path, image
	default:
		return
	}
	m.status = "Imagem selecionada. Salve o jogo para aplicar"
	invalid(m.hwnd)
}
func (m *managerState) selectExecutable(which string) {
	filter := "Executáveis (*.exe)\x00*.exe\x00Todos (*.*)\x00*.*\x00"
	if which == "Core" {
		filter = "Núcleos RetroArch (*.dll)\x00*.dll\x00Todos (*.*)\x00*.*\x00"
	}
	if which == "ROM" {
		filter = "Arquivos de jogos\x00*.sfc;*.smc;*.zip;*.nes;*.gb;*.gbc;*.gba;*.iso;*.chd;*.cue;*.n64;*.z64\x00Todos (*.*)\x00*.*\x00"
	}
	if which == "Exe" && m.mode != "retroarch" {
		filter = "Executáveis e atalhos\x00*.exe;*.lnk\x00Executáveis\x00*.exe\x00Atalhos\x00*.lnk\x00"
	}
	path := openFile(m.hwnd, filter, filepath.Dir(m.field(which)))
	if path != "" {
		m.set(which, path)
	}
}
func (m *managerState) startGameTest() {
	id := m.field("ID")
	if !validID.MatchString(id) {
		messageBox(m.hwnd, "Salve um ID válido antes de testar.", "Voyager")
		return
	}
	if _, err := os.Stat(configFile(id)); err != nil {
		messageBox(m.hwnd, "Salve o jogo antes de abrir a tela de loading real.", "Voyager")
		return
	}
	exe, _ := os.Executable()
	cmd := exec.Command(exe, "--game", id)
	cmd.Dir = rootDir
	if err := cmd.Start(); err != nil {
		messageBox(m.hwnd, err.Error(), "Teste do jogo")
	} else {
		_ = cmd.Process.Release()
		m.status = "Jogo iniciado pelo Voyager — " + id
	}
	invalid(m.hwnd)
}
func (m *managerState) capturePreview() {
	id := m.field("ID")
	if !validID.MatchString(id) {
		id = "Teste"
	}
	c := m.readUI()
	if c.Title == "" {
		c.Title = id
	}
	path, err := previewScreenshot(c, id, m.gridPath, m.heroPath)
	if err != nil {
		messageBox(m.hwnd, err.Error(), "Erro ao gerar o print")
		return
	}
	m.status = "Print 50% salvo em Imagens: " + filepath.Base(path)
	invalid(m.hwnd)
	messageBox(m.hwnd, "Imagem de teste gerada em 50%:\n\n"+path, "Voyager — Print de loading")
}
func (m *managerState) exploreFolder() {
	id := m.field("ID")
	if !validID.MatchString(id) {
		return
	}
	folder := gameDir(id)
	if _, err := os.Stat(folder); err != nil {
		messageBox(m.hwnd, "Salve o jogo primeiro.", "Voyager")
		return
	}
	call(pShellExecute, m.hwnd, ptrStr("open"), ptrStr(folder), 0, 0, SW_SHOWNORMAL)
}
func (m *managerState) importOld() {
	file := openFile(m.hwnd, "Configuração do jogo\x00config.json\x00Todos (*.*)\x00*.*\x00", rootDir)
	if file == "" {
		return
	}
	sourceGame := filepath.Dir(file)
	sourceRoot := filepath.Dir(sourceGame)
	if strings.EqualFold(sourceRoot, filepath.Join(rootDir, "Games")) {
		messageBox(m.hwnd, "Essa já é a pasta Games atual.", "Voyager")
		return
	}
	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		messageBox(m.hwnd, err.Error(), "Importação")
		return
	}
	count := 0
	for _, ent := range entries {
		if !ent.IsDir() || !validID.MatchString(ent.Name()) {
			continue
		}
		src := filepath.Join(sourceRoot, ent.Name())
		if _, err := os.Stat(filepath.Join(src, "config.json")); err != nil {
			continue
		}
		dst := gameDir(ent.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := duplicateDir(src, dst); err == nil {
			count++
		}
	}
	m.ids = listGames()
	invalid(m.hwnd)
	messageBox(m.hwnd, fmt.Sprintf("%d jogo(s) importado(s) sem substituir os existentes.", count), "Importação concluída")
}
func copyClipboard(hwnd uintptr, s string) {
	data := syscall.StringToUTF16(s)
	if call(u32.NewProc("OpenClipboard"), hwnd) == 0 {
		return
	}
	defer call(u32.NewProc("CloseClipboard"))
	call(u32.NewProc("EmptyClipboard"))
	size := uintptr(len(data) * 2)
	handle := call(k32.NewProc("GlobalAlloc"), 2, size)
	if handle == 0 {
		return
	}
	p := call(k32.NewProc("GlobalLock"), handle)
	if p == 0 {
		call(k32.NewProc("GlobalFree"), handle)
		return
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(data)), data)
	call(k32.NewProc("GlobalUnlock"), handle)
	if call(u32.NewProc("SetClipboardData"), 13, handle) == 0 {
		call(k32.NewProc("GlobalFree"), handle)
	}
}
func (m *managerState) click(x, y int) {
	if x < 250 && y >= 166 && y < m.windowH-225 {
		row := (y-168)/47 + m.scroll
		if row >= 0 && row < len(m.ids) {
			m.load(m.ids[row])
			return
		}
	}
	for _, b := range m.buttons {
		if b.has(x, y) {
			m.action(b.key)
			return
		}
	}
}
func (m *managerState) action(key string) {
	if m.settingsAction(key) { return }
	switch key {
	case "new":
		m.resetNew()
	case "import":
		m.importOld()
	case "save":
		if err := m.save(); err != nil {
			messageBox(m.hwnd, err.Error(), "Erro ao salvar")
		}
	case "launch":
		m.startGameTest()
	case "preview":
		m.capturePreview()
	case "folder":
		m.exploreFolder()
	case "hero":
		m.selectArt("hero")
	case "grid":
		m.selectArt("grid")
	case "cartridge":
		m.selectArt("cartridge")
	case "browseexe":
		m.selectExecutable("Exe")
	case "browsecore":
		m.selectExecutable("Core")
	case "browserom":
		m.selectExecutable("ROM")
	case "modera":
		m.mode = "retroarch"
		m.layout()
		invalid(m.hwnd)
	case "modewindows":
		m.mode = "executavel"
		m.layout()
		invalid(m.hwnd)
	case "copycommand":
		m.refreshCommands()
		copyClipboard(m.hwnd, getText(m.inputs["Command"]))
		m.status = "Comando Sunshine copiado"
		invalid(m.hwnd)
	case "copycover":
		m.refreshCommands()
		copyClipboard(m.hwnd, getText(m.inputs["Cover"]))
		m.status = "Caminho da capa copiado"
		invalid(m.hwnd)
	case "tab0", "tab1", "tab2", "tab3", "tab4":
		m.tab = int(key[3] - '0')
		m.layout()
		invalid(m.hwnd)
	}
}
func moveEdit(h uintptr, x, y, w, hh int, enabled bool) {
	if h == 0 {
		return
	}
	if enabled {
		call(pSetWindowPos, h, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(hh), SWP_NOACTIVATE|SWP_SHOWWINDOW)
		show(h, SW_SHOW)
	} else {
		show(h, SW_HIDE)
	}
}
func (m *managerState) layout() {
	if m.hwnd == 0 {
		return
	}
	r := client(m.hwnd)
	w, h := int(r.w()), int(r.h())
	if w < 1 || h < 1 {
		return
	}
	m.windowW = w
	m.windowH = h
	left := 310
	right := w - 35
	full := right - left
	half := (full - 25) / 2
	places := map[string]clickable{}
	add := func(k string, x, y, ww, hh int) { places[k] = clickable{k, x, y, ww, hh} }
	if m.tab == 0 {
		add("ID", left+8, 254, half-17, 33)
		add("Title", left+half+18, 254, half-18, 33)
		add("Platform", left+8, 353, half-17, 33)
		add("CoreName", left+half+18, 353, half-18, 33)
		add("Description", left+8, 457, full-20, 110)
	} else if m.tab == 2 {
		add("Exe", left+8, 320, full-153, 35)
		if m.mode == "retroarch" {
			add("Core", left+8, 414, full-153, 35)
		}
		add("ROM", left+8, 502, full-153, 35)
		if m.mode != "retroarch" {
			add("Process", left+8, 596, full-20, 35)
		}
	} else if m.tab == 3 {
		add("Command", left+8, 265, full-20, 72)
		add("Cover", left+8, 413, full-20, 35)
	}
	for key, handle := range m.inputs {
		r, ok := places[key]
		if ok {
			moveEdit(handle, r.x, r.y, r.w, r.h, true)
		} else {
			moveEdit(handle, 0, 0, 0, 0, false)
		}
	}
}
func (m *managerState) btn(p *painter, key, label string, x, y, w, h int, selected bool) {
	bg := cButton
	fg := cWhite
	if selected {
		bg = palette(appPrefs.DarkAccent())
	}
	p.fill(x, y, w, h, bg)
	p.fill(x, y+h-2, w, 2, 0xff45474f)
	tw := len([]rune(label)) * 7
	tx := x + (w-tw)/2
	if tx < x+8 {
		tx = x + 8
	}
	p.text(tx, y+(h-21)/2, w-16, 28, label, 12, fg, true)
	m.buttons = append(m.buttons, clickable{key, x, y, w, h})
}
func (m *managerState) paint(p *painter) {
	w, h := m.windowW, m.windowH
	m.buttons = m.buttons[:0]
	p.clear(cBG)
	p.fill(0, 0, 270, h, cSide)
	p.fill(269, 0, 1, h, cEdge)
	p.text(25, 25, 220, 43, "VOYAGER", 29, cWhite, true)
	p.text(27, 72, 220, 25, "MOONLIGHT LOADING  •  v"+version, 10, cMuted, true)
	p.text(27, 132, 190, 25, "MEUS JOGOS", 12, 0xffbbbbc2, true)
	p.text(229, 132, 35, 25, fmt.Sprintf("%d", len(m.ids)), 12, cMuted, true)
	maxRows := (h - 405) / 47
	if maxRows < 1 {
		maxRows = 1
	}
	if m.scroll > len(m.ids)-maxRows {
		m.scroll = len(m.ids) - maxRows
		if m.scroll < 0 {
			m.scroll = 0
		}
	}
	for row := 0; row < maxRows && row+m.scroll < len(m.ids); row++ {
		id := m.ids[row+m.scroll]
		y := 168 + 47*row
		if id == m.activeID {
			p.fill(17, y, 237, 42, palette(appPrefs.DarkAccent()))
		} else {
			p.fill(17, y, 237, 42, 0xff1c1c20)
		}
		p.text(30, y+10, 210, 28, id, 13, cWhite, id == m.activeID)
	}
	m.btn(p, "settings", "CONFIGURAÇÕES", 23, h-209, 225, 42, m.tab == 4)
	m.btn(p, "new", "+  NOVO JOGO", 23, h-151, 225, 42, false)
	m.btn(p, "import", "IMPORTAR JOGOS", 23, h-98, 225, 40, false)
	p.fill(22, 24, 4, 42, palette(appPrefs.Accent()))
	p.text(310, 26, w-340, 24, "BIBLIOTECA DE JOGOS", 11, cMuted, true)
	title := m.field("Title")
	if m.tab == 4 {
        title = "Configurações do Voyager"
    } else if title == "" {
        title = "Configuração do jogo"
    }
	p.text(310, 54, w-420, 54, title, 28, cWhite, true)
	p.fill(w-269, 32, 229, 36, 0xff232329)
	p.text(w-250, 39, 207, 23, "SUNSHINE  •  MOONLIGHT", 10, cMuted, true)
	tabNames := []string{"JOGO", "IMAGENS", "EXECUÇÃO", "SUNSHINE", "AJUSTES"}
	x := 310
	for i, t := range tabNames {
		ww := 110
		if i == 2 {
			ww = 135
		}
		if i == 3 {
			ww = 124
		}
        if i == 4 { ww = 110 }
		selected := m.tab == i
		if selected {
			p.fill(x, 128, ww, 40, 0xff34343a)
			p.fill(x, 164, ww, 3, palette(appPrefs.Accent()))
		} else {
			p.fill(x, 128, ww, 40, 0xff19191d)
		}
		p.text(x+18, 137, ww-18, 24, t, 12, cWhite, selected)
		m.buttons = append(m.buttons, clickable{fmt.Sprintf("tab%d", i), x, 128, ww, 40})
		x += ww + 5
	}
	p.fill(310, 174, w-340, h-264, cPanel)
	label := func(x, y int, s string) { p.text(x, y, 800, 22, s, 11, cMuted, true) }
	section := func(s string) { p.text(331, 186, w-400, 39, s, 20, cWhite, true) }
	fieldRect := func(x, y, ww, hh int) { p.fill(x-3, y-3, ww+6, hh+6, cEdge); p.fill(x, y, ww, hh, cField) }
	left, full := 310, w-345
	half := (full - 25) / 2
	switch m.tab {
	case 0:
		section("Identidade do jogo")
		label(left+8, 228, "ID DA PASTA")
		label(left+half+18, 228, "TÍTULO")
		fieldRect(left+8, 254, half-17, 33)
		fieldRect(left+half+18, 254, half-18, 33)
		label(left+8, 329, "PLATAFORMA")
		label(left+half+18, 329, "NÚCLEO / EMULADOR")
		fieldRect(left+8, 353, half-17, 33)
		fieldRect(left+half+18, 353, half-18, 33)
		label(left+8, 432, "DESCRIÇÃO DO JOGO")
		fieldRect(left+8, 457, full-20, 110)
		p.text(left+8, 581, full-20, 38, "A tela de loading termina 1 segundo depois da janela do jogo ficar visível.", 12, cMuted, false)
	case 1:
		section("Identidade visual do jogo")
		p.text(331, 224, w-400, 36, "HERO: fundo do loading  •  GRID: capa do Moonlight  •  CARTRIDGE: reservado.", 12, cMuted, false)
		gap := 14
		colWidth := (full - 16 - 2*gap) / 3
		for i, item := range []struct {
			key, title, button, path, hint string
			image *img
		}{
			{"hero", "HERO  •  1920 x 620", "SELECIONAR HERO", m.heroPath, "Fundo do loading", m.hero},
			{"grid", "GRID  •  600 x 900", "SELECIONAR GRID", m.gridPath, "Capa para o Moonlight", m.grid},
			{"cartridge", "CARTRIDGE  •  1536 x 1024", "SELECIONAR CARTRIDGE", m.cartridgePath, "Ainda não exibido no loading", m.cartridge},
		} {
			x := left + 8 + i*(colWidth+gap)
			label(x, 263, item.title)
			p.fill(x, 299, colWidth, 228, cField)
			p.imageContain(item.image, x+4, 303, colWidth-8, 220)
			m.btn(p, item.key, item.button, x, 540, colWidth, 44, false)
			if item.path != "" {
				p.text(x, 591, colWidth, 32, filepath.Base(item.path), 11, cMuted, false)
			} else {
				p.text(x, 591, colWidth, 32, "Nenhuma imagem selecionada", 11, cMuted, false)
			}
			p.text(x, 632, colWidth, 35, item.hint, 11, cMuted, false)
		}
	case 2:
		section("Como iniciar o jogo")
		label(left+8, 222, "TIPO DE EXECUÇÃO")
		m.btn(p, "modera", "RETROARCH", left+8, 251, 180, 38, m.mode == "retroarch")
		m.btn(p, "modewindows", "JOGO WINDOWS / ATALHO", left+198, 251, 227, 38, m.mode != "retroarch")
		execLabel := "CAMINHO DO RETROARCH.EXE"
		if m.mode != "retroarch" {
			execLabel = "EXECUTÁVEL (.EXE) OU ATALHO (.LNK)"
		}
		label(left+8, 296, execLabel)
		fieldRect(left+8, 320, full-153, 35)
		m.btn(p, "browseexe", "PROCURAR", w-171, 320, 125, 35, false)
		if m.mode == "retroarch" {
			label(left+8, 390, "NÚCLEO DLL DO RETROARCH")
			fieldRect(left+8, 414, full-153, 35)
			m.btn(p, "browsecore", "PROCURAR", w-171, 414, 125, 35, false)
			label(left+8, 476, "ROM DO JOGO")
			fieldRect(left+8, 502, full-153, 35)
			m.btn(p, "browserom", "PROCURAR", w-171, 502, 125, 35, false)
		} else {
			label(left+8, 476, "ARGUMENTOS ADICIONAIS (OPCIONAL)")
			fieldRect(left+8, 502, full-153, 35)
			label(left+8, 570, "PROCESSO DO JOGO (OPCIONAL)")
			fieldRect(left+8, 596, full-20, 35)
			p.text(left+8, 635, full-24, 31, "Para launchers que abrem outro executável, informe o nome do processo real.", 11, cMuted, false)
		}
	case 4:
        m.paintSettings(p)
    case 3:
        section("Conectar ao Moonlight")
		label(left+8, 235, "COMANDO DE INICIALIZAÇÃO")
		fieldRect(left+8, 265, full-20, 72)
		m.btn(p, "copycommand", "COPIAR COMANDO COMPLETO", left+8, 348, 285, 42, false)
		label(left+8, 386, "CAMINHO ABSOLUTO DA CAPA GRID")
		fieldRect(left+8, 413, full-20, 35)
		m.btn(p, "copycover", "COPIAR CAMINHO DA CAPA", left+8, 461, 275, 42, false)
		p.fill(left+8, 534, full-20, 105, 0xff232329)
		p.text(left+24, 548, full-60, 28, "Configuração sugerida para o Sunshine", 14, cWhite, true)
		p.text(left+24, 581, full-70, 55, "Adicione o comando em Detached Commands. Deixe o Command vazio para a sessão continuar após o loading.", 12, cMuted, false)
	}
	p.fill(271, h-84, w-271, 84, 0xff101013)
	p.text(310, h-34, w-720, 28, m.status, 11, cMuted, false)
	bw := 135
	gap := 9
	right := w - 24
	if m.tab == 4 {
        m.paintSettingsFooter(p, right, h)
    } else {
        m.btn(p, "save", "SALVAR", right-(bw+gap)*4, h-72, bw, 44, true)
        m.btn(p, "launch", "TESTAR JOGO", right-(bw+gap)*3, h-72, bw, 44, false)
        m.btn(p, "preview", "PRINT 50%", right-(bw+gap)*2, h-72, bw, 44, false)
        m.btn(p, "folder", "ABRIR PASTA", right-(bw+gap), h-72, bw, 44, false)
    }
}
