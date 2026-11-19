//go:build windows

package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"voyager/native/internal/appearance"
	"voyager/native/internal/heroaccent"
	ll "voyager/native/internal/loadinglayout"
)

// Native Win32 overlay. No PowerShell, Windows Forms, browser or .NET runtime.
var overlayTheme = appearance.Default()

func loadOverlayTheme() {
    overlayTheme = appearance.Load(filepath.Join(rootDir, "themes", "default.json"))
    overlayTheme.Colors["dimOverlay"] = fmt.Sprintf("#%02X080A0C", appPrefs.BackgroundDim*255/100)
}

type overlayState struct {
	cfg                                gameConfig
	id                                 string
	preview                            bool
	windows                            []uintptr
	cartridge, background              *img
	accent                             uint32
	layout                             ll.Layout
	buffers                            map[uintptr]*overlayBackbuffer
	percent                            int
	phase, detail                      string
	started, ready, fadeAt, timeFailed time.Time
	targetHWND                         uintptr
	launchPID                          uint32
	mainExe, launchDir                 string
	process                            *os.Process
	baseline                           map[uint32]bool
	lastCheck, lastTopmost             time.Time
	fail                               string
	shown                              bool
}

var overlay *overlayState
var overlayCallback = syscall.NewCallback(overlayWndProc)

type windowCandidate struct {
	handle uintptr
	pid    uint32
}

var enumeratedWindows []windowCandidate

// Reuse the callback; allocating a syscall callback on every detection tick would leak trampolines.
var enumGameWindowCallback = syscall.NewCallback(func(hwnd, param uintptr) uintptr {
	var pid uint32
	call(pGetWindowThreadProcessId, hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid != 0 && isUsableWindow(hwnd) {
		enumeratedWindows = append(enumeratedWindows, windowCandidate{hwnd, pid})
	}
	return 1
})
var overlayClassName = syscall.StringToUTF16Ptr("VoyagerLoadingNativeV6")

func overlayWndProc(hwnd uintptr, m uint32, wp, lp uintptr) uintptr {
	switch m {
	case WM_PAINT:
		var ps paintStruct
		hdc := call(pBeginPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
		if overlay != nil {
			rc := client(hwnd)
			// Present the complete frame instead of showing each redraw operation.
			overlay.paintBuffered(hwnd,hdc,int(rc.w()),int(rc.h()))
		}
		call(pEndPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_TIMER:
		if overlay != nil && len(overlay.windows) > 0 && hwnd == overlay.windows[0] {
			overlay.tick()
		}
		return 0
	case WM_CLOSE:
		call(pDestroyWindow, hwnd)
		return 0
	case WM_DESTROY:
		if overlay != nil {overlay.dropBuffer(hwnd)}
		// The overlay group must close together; caller exits its dedicated message loop.
		if overlay != nil && len(overlay.windows) > 0 && hwnd == overlay.windows[0] {
			call(pPostQuitMessage, 0)
		}
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_KEYDOWN:
		if wp == VK_ESCAPE && overlay != nil && overlay.preview {
			overlay.stop()
		}
		return 0
	}
	return call(pDefWindowProc, hwnd, uintptr(m), wp, lp)
}
func drawOverlay(p *painter, c gameConfig, originX, originY, w, h, percent int, phase, detail string, cartridge, background *img, heroColor uint32, bounceY float64, layout ll.Layout, failed bool) {
    if w<1 || h<1 {return}
    p.fill(originX,originY,w,h,palette(overlayTheme.Color("background")))
    if background!=nil {p.imageCover(background,originX,originY,w,h)}
    p.fill(originX,originY,w,h,palette(overlayTheme.Color("dimOverlay")))
    s:=math.Min(float64(w)/ll.VirtualWidth,float64(h)/ll.VirtualHeight)
    if s>2.6{s=2.6}
    if s<=0{return}
    left:=float64(originX)+(float64(w)-1600*s)/2
    top:=float64(originY)+(float64(h)-900*s)/2
    X:=func(v float64)int{return int(math.Round(left+v*s))}
    Y:=func(v float64)int{return int(math.Round(top+v*s))}
    D:=func(v float64)int{return int(math.Max(1,math.Round(v*s)))}
    if heroColor==0 {heroColor=heroaccent.Neutral}
    if phase==""{phase="Preparando o jogo"}
    if failed{phase=detail}
    description:=[]rune(c.Description)
    if len(description)>270{description=description[:270]}
    for _,n:=range layout.Elements {
        if !n.Visible && !(failed && n.ID==ll.Phase){continue}
        xx,yy,ww,hh:=X(n.X),Y(n.Y),D(n.W),D(n.H)
        if n.ID==ll.Cartridge {
            if cartridge!=nil {p.imageContain(cartridge,xx,Y(n.Y+bounceY),ww,hh)}
            continue
        }
        if n.ID==ll.Progress {
            height:=hh
            if height<2 {height=2}
            p.fill(xx,yy,ww,height,palette(overlayTheme.Color("progressTrack")))
            progressColor:=palette(heroColor)
            if failed{progressColor=palette(overlayTheme.Color("progressError"))}
            clamped:=percent
            if clamped<0{clamped=0}
            if clamped>100{clamped=100}
            if clamped>0{p.fill(xx,yy,int(float64(ww)*float64(clamped)/100),height,progressColor)}
            continue
        }
        var value string
        var themeColor uint32
        bold:=false
        switch n.ID {
        case ll.Platform:
            value=strings.ToUpper(c.Platform)
            themeColor=overlayTheme.Color("platform")
        case ll.Title:
            value=c.Title
            themeColor=overlayTheme.Color("gameTitle")
            bold=true
        case ll.Description:
            if !appPrefs.ShowDescription{continue}
            value=string(description)
            themeColor=overlayTheme.Color("description")
        case ll.Heading:
            value=overlayTheme.LoadingTitle
            themeColor=overlayTheme.Color("heading")
            if failed{value=overlayTheme.FailureTitle;themeColor=overlayTheme.Color("headingError")}
        case ll.Percentage:
            if !appPrefs.ShowPercentage{continue}
            value=fmt.Sprintf("%d%%",percent)
            themeColor=overlayTheme.Color("percentage")
        case ll.Phase:
            if !appPrefs.ShowPhase && !failed{continue}
            value=phase
            themeColor=overlayTheme.Color("status")
        default:continue
        }
        switch n.Color {
        case "white":themeColor=overlayTheme.Color("gameTitle")
        case "muted":themeColor=overlayTheme.Color("status")
        case "hero":themeColor=heroColor
        }
        if failed && (n.ID==ll.Heading || n.ID==ll.Phase) {
            themeColor=overlayTheme.Color("headingError")
        }
        p.text(xx,yy,ww,hh,value,n.FontSize*float32(s),palette(themeColor),bold)
    }
}
func loadBlurry(src string) *img {
	if src == "" {
		return nil
	}
	stat, err := os.Stat(src)
	if err != nil {
		return loadImg(src)
	}
	cache := filepath.Join(os.TempDir(), "VoyagerLauncher", "blur")
	_ = os.MkdirAll(cache, 0700)
	name := fmt.Sprintf("%x-%d-%d.png", simpleHash(src), stat.Size(), stat.ModTime().UnixNano())
	dst := filepath.Join(cache, name)
	if _, e := os.Stat(dst); e == nil {
		return loadImg(dst)
	}
	f, err := os.Open(src)
	if err != nil {
		return loadImg(src)
	}
	original, _, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return loadImg(src)
	}
	w, h := 288, 162
	bounds := original.Bounds()
	sw, sh := float64(bounds.Dx()), float64(bounds.Dy())
	// Match the screen 16:9 aspect ratio, then blur two separable passes.
	aspect := float64(w) / float64(h)
	if sw/sh > aspect {
		sw = sh * aspect
	} else {
		sh = sw / aspect
	}
	offx := (float64(bounds.Dx()) - sw) / 2
	offy := (float64(bounds.Dy()) - sh) / 2
	pix := make([]color.NRGBA, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			px := bounds.Min.X + int(offx+(float64(x)+.5)*sw/float64(w))
			py := bounds.Min.Y + int(offy+(float64(y)+.5)*sh/float64(h))
			r, g, b, _ := original.At(px, py).RGBA()
			pix[y*w+x] = color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}
		}
	}
	tmp := make([]color.NRGBA, len(pix))
	for pass := 0; pass < 2; pass++ {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, n := 0, 0, 0, 0
				for dx := -5; dx <= 5; dx++ {
					xx := x + dx
					if xx < 0 {
						xx = 0
					}
					if xx >= w {
						xx = w - 1
					}
					c := pix[y*w+xx]
					r += int(c.R)
					g += int(c.G)
					b += int(c.B)
					n++
				}
				tmp[y*w+x] = color.NRGBA{uint8(r / n), uint8(g / n), uint8(b / n), 255}
			}
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, n := 0, 0, 0, 0
				for dy := -5; dy <= 5; dy++ {
					yy := y + dy
					if yy < 0 {
						yy = 0
					}
					if yy >= h {
						yy = h - 1
					}
					c := tmp[yy*w+x]
					r += int(c.R)
					g += int(c.G)
					b += int(c.B)
					n++
				}
				pix[y*w+x] = color.NRGBA{uint8(r / n), uint8(g / n), uint8(b / n), 255}
			}
		}
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.SetNRGBA(x, y, pix[y*w+x])
		}
	}
	wf, err := os.Create(dst)
	if err == nil {
		err = png.Encode(wf, out)
		_ = wf.Close()
	}
	if err != nil {
		return loadImg(src)
	}
	return loadImg(dst)
}
func simpleHash(s string) uint64 {
	var h uint64 = 14695981039346656037
	for _, b := range []byte(s) {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}

func previewScreenshot(c gameConfig, id, gridPath, heroPath, cartridgePath string) (string, error) {
	// A newly selected Cartridge can be previewed even before saving the game.
	// Older games fall back to their Grid cover until a Cartridge is configured.
	cartridge := loadImg(cartridgePath)
	if cartridge == nil {
		cartridge = loadImg(gridPath)
	}
	if cartridge != nil {
		defer cartridge.close()
	}
	var bg *img
    if appPrefs.BlurBackground { bg = loadBlurry(heroPath) } else { bg = loadImg(heroPath) }
	if bg != nil {
		defer bg.close()
	}
	p, bmp, err := newBitmapGraphics(1920, 1080)
	if err != nil {
		return "", err
	}
	layout,_:=loadEffectiveLoadingLayout(id)
	drawOverlay(p, c, 0, 0, 1920, 1080, 50, "Preparando a tela do jogo", "", cartridge, bg, heroaccent.FromFile(heroPath), 0, layout, false)
	p.close()
	defer call(pGdipDisposeImage, bmp)
	folder := knownPictures()
	if folder == "" {
		home, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		folder = filepath.Join(home, "Pictures")
	}
	dst := filepath.Join(folder, screenshotName(id))
	return dst, savePNG(bmp, dst)
}

func registerOverlayWindow() error {
	instance := call(pGetModuleHandle, 0)
	wc := wndClass{Size: uint32(unsafe.Sizeof(wndClass{})), Style: CS_HREDRAW | CS_VREDRAW, WndProc: overlayCallback, Instance: instance, Cursor: call(pLoadCursor, 0, IDC_ARROW), ClassName: uintptr(unsafe.Pointer(overlayClassName))}
	if call(pRegisterClass, uintptr(unsafe.Pointer(&wc))) == 0 {
		return winError("não foi possível registrar janela do loading")
	}
	return nil
}
func screenRects() []rect {
	result := []rect{}
	cb := syscall.NewCallback(func(hMon, hdc, lpRect, lparam uintptr) uintptr {
		var inf monitorInfo
		inf.Size = uint32(unsafe.Sizeof(inf))
		if call(pGetMonitorInfo, hMon, uintptr(unsafe.Pointer(&inf))) != 0 {
			result = append(result, inf.Monitor)
		}
		return 1
	})
	call(pEnumDisplayMonitors, 0, 0, cb, 0)
	if len(result) == 0 {
		result = append(result, rect{0, 0, int32(call(pGetSystemMetrics, SM_CXSCREEN)), int32(call(pGetSystemMetrics, SM_CYSCREEN))})
	}
	return result
}
// A subtle, continuous vertical bob: roughly 8 reference pixels, 2.6 s per cycle.
func (o *overlayState) cartridgeOffset() float64 {
	if o.started.IsZero() { return 0 }
	// Sinusoidal easing avoids abrupt direction changes. The static background
	// and text are cached, so only the cartridge is animated at up to ~60 FPS.
	return 11 * math.Sin(time.Since(o.started).Seconds() * (2 * math.Pi / 2.9))
}

func (o *overlayState) stop() {
	if len(o.windows) == 0 {
		return
	}
	for _, hwnd := range o.windows[1:] {
		call(pDestroyWindow, hwnd)
	}
	call(pKillTimer, o.windows[0], 1)
	call(pDestroyWindow, o.windows[0])
	o.windows = nil
}
func (o *overlayState) invalidateAll() {
	for _, h := range o.windows {
		invalid(h)
	}
}
func (o *overlayState) run() error {
	o.layout,_ = loadEffectiveLoadingLayout(o.id)
	overlay = o
	defer func() {
		o.freeBuffers()
		overlay = nil
		if o.cartridge != nil {
			o.cartridge.close()
		}
		if o.background != nil {
			o.background.close()
		}
	}()
	// The loaded cartridge is per-game; keep Grid as a fallback for older configs.
	o.cartridge = loadImg(artworkPath(o.cfg, o.id, "cartridge"))
	if o.cartridge == nil {
		o.cartridge = loadImg(artworkPath(o.cfg, o.id, "grid"))
	}
	heroPath := artworkPath(o.cfg, o.id, "hero")
	// Cor calculada uma unica vez, a partir da arte original do jogo.
	// O tema escolhido em AJUSTES afeta somente a interface do gerenciador.
	o.accent = heroaccent.FromFile(heroPath)
	if appPrefs.BlurBackground {
		o.background = loadBlurry(heroPath)
	} else {
		o.background = loadImg(heroPath)
	}
	if err := registerOverlayWindow(); err != nil {
		return err
	}
	screens := screenRects()
	instance := call(pGetModuleHandle, 0)
	for _, r := range screens {
		h := call(pCreateWindow, WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_NOACTIVATE|WS_EX_LAYERED, uintptr(unsafe.Pointer(overlayClassName)), ptrStr(""), WS_POPUP|WS_VISIBLE, uintptr(r.Left), uintptr(r.Top), uintptr(r.w()), uintptr(r.h()), 0, 0, instance, 0)
		if h == 0 {
			continue
		}
		o.windows = append(o.windows, h)
		call(pSetLayeredWindowAttributes, h, 0, 255, LWA_ALPHA)
		show(h, SW_SHOW)
		call(pUpdateWindow, h)
	}
	if len(o.windows) == 0 {
		return winError("não foi possível criar tela cheia")
	}
	o.started = time.Now()
	o.percent = 5
	o.phase = "Verificando arquivos do jogo"
	o.baseline = snapshotPIDs()
	o.invalidateAll()
	if err := o.startGame(); err != nil {
		o.fail = err.Error()
		o.timeFailed = time.Now()
		o.phase = o.fail
		o.invalidateAll()
	}
	// Request ~60 FPS while the loading is active; Windows may coalesce ticks.
	// Detection stays throttled separately at ~160ms.
	call(pSetTimer, o.windows[0], 1, 16, 0)
	windowsLoop()
	return nil
}
func (o *overlayState) startGame() error {
	c := o.cfg
	mode := strings.ToLower(c.Mode)
	if mode == "" {
		mode = "retroarch"
	}
	var execPath, workdir string
	var args []string
	if mode == "retroarch" {
		execPath = resolveGameFile(c.RetroarchPath, gameDir(o.id))
		core := resolveGameFile(c.CorePath, gameDir(o.id))
		rom := resolveGameFile(c.RomPath, gameDir(o.id))
		for _, p := range []string{execPath, core, rom} {
			if _, err := os.Stat(p); err != nil {
				return fmt.Errorf("arquivo não encontrado: %s", p)
			}
		}
		workdir = filepath.Dir(execPath)
		for _, p := range snapshotProcesses() {
			if p.name == "retroarch.exe" {
				return fmt.Errorf("feche o RetroArch que já está aberto antes de iniciar outro jogo")
			}
		}
		config := filepath.Join(workdir, "retroarch.cfg")
		if _, err := os.Stat(config); err == nil {
			args = append(args, "--config", config)
		}
		args = append(args, "-L", core, rom, "-f")
	} else if mode == "executavel" || mode == "atalho" {
		execPath = c.ExecutablePath
		if execPath == "" {
			execPath = c.ShortcutPath
		}
		execPath = resolveGameFile(execPath, gameDir(o.id))
		if _, err := os.Stat(execPath); err != nil {
			return fmt.Errorf("executável não encontrado: %s", execPath)
		}
		workdir = expandWindows(c.WorkingDirectory)
		if workdir == "" {
			workdir = filepath.Dir(execPath)
		}
		if c.Arguments != "" {
			args = splitWindowsArgs(c.Arguments)
		}
	} else {
		return fmt.Errorf("modo não suportado: %s", mode)
	}
	o.mainExe = execPath
	o.launchDir = workdir
	o.percent = 25
	o.phase = "Abrindo o jogo"
	o.invalidateAll()
	if strings.EqualFold(filepath.Ext(execPath), ".lnk") {
		// ShellExecute resolves Windows shortcuts. The process tracker finds the actual game window.
		verb := ptrStr("open")
		r := call(pShellExecute, 0, verb, ptrStr(execPath), ptrStr(c.Arguments), ptrStr(workdir), SW_SHOWNORMAL)
		if r <= 32 {
			return fmt.Errorf("não foi possível abrir o atalho (%d)", r)
		}
	} else {
		cmd := exec.Command(execPath, args...)
		cmd.Dir = workdir
		if err := cmd.Start(); err != nil {
			return err
		}
		o.process = cmd.Process
		o.launchPID = uint32(cmd.Process.Pid)
		// Avoid a zombie / leaked child handle, but the executable does not wait for game exit.
		_ = cmd.Process.Release()
	}
	o.percent = 45
	o.phase = "Aguardando o jogo abrir"
	o.started = time.Now()
	o.invalidateAll()
	return nil
}

// Basic Windows argument quoting. Arguments are not passed through CMD or PowerShell.
func splitWindowsArgs(s string) []string {
	var out []string
	var b strings.Builder
	quoted := false
	slashes := 0
	active := false
	flush := func() {
		if active {
			out = append(out, b.String())
			b.Reset()
			active = false
		}
	}
	for _, ch := range s {
		if ch == '\\' {
			slashes++
			active = true
			continue
		}
		if ch == '"' {
			for i := 0; i < slashes/2; i++ {
				b.WriteByte('\\')
			}
			if slashes%2 == 1 {
				b.WriteByte('"')
			} else {
				quoted = !quoted
			}
			slashes = 0
			active = true
			continue
		}
		for i := 0; i < slashes; i++ {
			b.WriteByte('\\')
		}
		slashes = 0
		if (ch == ' ' || ch == '\t') && !quoted {
			flush()
		} else {
			b.WriteRune(ch)
			active = true
		}
	}
	for i := 0; i < slashes; i++ {
		b.WriteByte('\\')
	}
	flush()
	return out
}

type procInfo struct {
	pid, parent uint32
	name, path  string
}

func snapshotProcesses() map[uint32]procInfo {
	result := make(map[uint32]procInfo)
	h := call(pCreateSnapshot, TH32CS_SNAPPROCESS, 0)
	if h == ^uintptr(0) {
		return result
	}
	defer call(pCloseHandle, h)
	var e processEntry
	e.Size = uint32(unsafe.Sizeof(e))
	if call(pProcessFirst, h, uintptr(unsafe.Pointer(&e))) == 0 {
		return result
	}
	for {
		name := utf16zero(e.ExeFile[:])
		result[e.ProcessID] = procInfo{pid: e.ProcessID, parent: e.ParentID, name: strings.ToLower(name)}
		if call(pProcessNext, h, uintptr(unsafe.Pointer(&e))) == 0 {
			break
		}
	}
	return result
}
func snapshotPIDs() map[uint32]bool {
	m := make(map[uint32]bool)
	for pid := range snapshotProcesses() {
		m[pid] = true
	}
	return m
}
func procPath(pid uint32) string {
	h := call(pOpenProcess, PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer call(pCloseHandle, h)
	buf := make([]uint16, 2048)
	length := uint32(len(buf))
	if call(pQueryFullProcessImageName, h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&length))) == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:int(length)])
}
func childOf(pid, parent uint32, procs map[uint32]procInfo) bool {
	if parent == 0 || pid == 0 {
		return false
	}
	for i := 0; i < 12; i++ {
		if pid == parent {
			return true
		}
		p, ok := procs[pid]
		if !ok || p.parent == 0 || p.parent == pid {
			return false
		}
		pid = p.parent
	}
	return false
}
func isUsableWindow(h uintptr) bool {
	if call(pIsWindowVisible, h) == 0 || call(pIsIconic, h) != 0 {
		return false
	}
	var cloaked uint32
	if call(pDwmGet, h, DWMWA_CLOAKED, uintptr(unsafe.Pointer(&cloaked)), 4) == 0 && cloaked != 0 {
		return false
	}
	var r rect
	call(pGetWindowRect, h, uintptr(unsafe.Pointer(&r)))
	if r.w() < 220 || r.h() < 125 {
		return false
	}
	// Not requiring a caption: borderless / maximized game windows often have no title.
	return true
}
func (o *overlayState) findGameWindow() uintptr {
	ps := snapshotProcesses()
	best, bestScore := uintptr(0), -1
	requested := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(o.cfg.WaitForProcess), ".exe"))
	enumeratedWindows = enumeratedWindows[:0]
	call(pEnumWindows, enumGameWindowCallback, 0)
	fallback := []uintptr{}
	for _, w := range enumeratedWindows {
		pi, ok := ps[w.pid]
		if !ok {
			continue
		}
		name := strings.TrimSuffix(strings.ToLower(pi.name), ".exe")
		if w.pid == uint32(os.Getpid()) || strings.Contains(name, "voyager") || strings.Contains(name, "sunshine") || name == "explorer" || name == "dwm" || name == "steamwebhelper" || name == "conhost" {
			continue
		}
		newProcess := !o.baseline[w.pid]
		if newProcess && requested == "" {
			fallback = append(fallback, w.handle)
		}
		score := 0
		if w.pid == o.launchPID && o.launchPID != 0 {
			score = 180
		}
		if childOf(w.pid, o.launchPID, ps) && o.launchPID != 0 && score < 135 {
			score = 135
		}
		if requested != "" {
			if name != requested {
				continue
			}
			if newProcess {
				score += 150
			} else {
				score += 20
			}
		}
		if newProcess {
			score += 24
		}
		path := procPath(w.pid)
		if path != "" {
			if strings.EqualFold(path, o.mainExe) {
				score += 85
			}
			if o.launchDir != "" && strings.HasPrefix(strings.ToLower(path), strings.ToLower(filepath.Clean(o.launchDir)+string(os.PathSeparator))) {
				score += 55
			}
		}
		if score > bestScore && score >= 55 {
			best = w.handle
			bestScore = score
		}
	}
	// If a shortcut/Steam launcher detached the real game, prefer a single newly
	// created visible window rather than accidentally tracking an existing app.
	if best == 0 && len(fallback) == 1 && time.Since(o.started) > 1200*time.Millisecond {
		return fallback[0]
	}
	return best
}
func (o *overlayState) tick() {
	now := time.Now()
	if o.fail != "" {
		if now.Sub(o.timeFailed) > 6*time.Second {
			o.stop()
		}
		return
	}
	if !o.fadeAt.IsZero() {
		age := now.Sub(o.fadeAt)
		level := 255 - int(math.Min(1, age.Seconds()/.20)*255)
		for _, hw := range o.windows {
			call(pSetLayeredWindowAttributes, hw, 0, uintptr(level), LWA_ALPHA)
		}
		if age >= 200*time.Millisecond {
			o.stop()
		}
		return
	}
	// Refresh the floating Cartridge even when the loading percentage is unchanged.
	if o.cartridge != nil {
		o.invalidateAll()
	}
	// Do not call SetWindowPos at every animation frame: this can stall
	// the compositor and create microstutter on fullscreen games.
	if now.Sub(o.lastTopmost)>=250*time.Millisecond {
		o.lastTopmost=now
		for _,hw:=range o.windows {
			call(pSetWindowPos, hw, HWND_TOPMOST, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE)
		}
	}
	if now.Sub(o.lastCheck) < 160*time.Millisecond {
		return
	}
	o.lastCheck = now
	candidate := o.findGameWindow()
	if candidate != 0 {
		if o.targetHWND != candidate {
			o.targetHWND = candidate
			o.ready = now
			o.percent = 70
			o.phase = "Janela do jogo detectada"
			o.invalidateAll()
		}
		age := now.Sub(o.ready)
		// Hold only after the same game window is continuously detected.
		// No artificial delay is added before opening the game.
		hold := time.Duration(appPrefs.PostOpenSeconds) * time.Second
		if hold < time.Second { hold = time.Second }
		if age >= hold {
			o.percent = 100
			o.phase = "Jogo pronto"
			o.invalidateAll()
			call(pSetForegroundWindow, candidate)
			o.fadeAt = now
		} else {
			v := 70 + int(age.Seconds()/hold.Seconds()*25)
			if v > 95 {
				v = 95
			}
			if v != o.percent {
				o.percent = v
				o.phase = "Confirmando imagem do jogo"
				o.invalidateAll()
			}
		}
	} else {
		if o.targetHWND != 0 {
			o.targetHWND = 0
			o.ready = time.Time{}
			o.percent = 55
			o.phase = "Aguardando a janela do jogo"
			o.invalidateAll()
		}
		if o.percent < 55 {
			o.percent = 55
			o.phase = "Aguardando a janela do jogo"
			o.invalidateAll()
		}
	}
	if now.Sub(o.started) > 120*time.Second {
		o.fail = "O jogo não apresentou uma janela identificável. Confira o executável ou informe o processo do jogo na aba Execução."
		o.timeFailed = now
		o.percent = 0
		o.invalidateAll()
	}
}

// Registration for JPEG in image.Decode, besides PNG and GIF.
