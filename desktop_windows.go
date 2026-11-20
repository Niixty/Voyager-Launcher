package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/go-webview2/webviewloader"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wr "github.com/wailsapp/wails/v2/pkg/runtime"
	"voyager/native/internal/heroaccent"
	ll "voyager/native/internal/loadinglayout"
	"voyager/native/internal/settings"
)

//go:embed app/assets/VoyagerLauncher.ico
var brandIcon []byte

//go:embed desktop/*
var web embed.FS

type Studio struct {
	ctx context.Context
	mu  sync.Mutex
}
type Entry struct {
	ID      string
	Config  gameConfig
	Problem string
}

func newStudio() error {
	if v, e := webviewloader.GetAvailableCoreWebView2BrowserVersionString(""); e != nil || v == "" {
		return errors.New("WebView2 indisponível; abrindo o gerenciador clássico")
	}
	assets, err := fs.Sub(web, "desktop")
	if err != nil {
		return err
	}
	s := &Studio{}
	return wails.Run(studioOptions(s, assets))
}
func studioOptions(s *Studio, assets fs.FS) *options.App {
	return &options.App{Title: "Voyager Launcher", Width: 1240, Height: 820, MinWidth: 700, MinHeight: 520,
		BackgroundColour: options.NewRGB(18, 20, 25), AssetServer: &assetserver.Options{Assets: assets, Handler: http.HandlerFunc(serveArtwork)}, Bind: []interface{}{s},
		OnStartup: func(ctx context.Context) {
			s.ctx = ctx
			go func() { _, err := probeUpdateStatus(); logSilentUpdateError(err) }()
		},
		OnShutdown: func(context.Context) { logSilentUpdateError(launchPendingUpdate()) }}
}
func artKind(k string) bool { return k == "hero" || k == "grid" || k == "cartridge" }
func serveArtwork(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/icon.ico" {
		w.Header().Set("Content-Type", "image/x-icon")
		http.ServeContent(w, r, "icon.ico", time.Time{}, bytes.NewReader(brandIcon))
		return
	}
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) != 3 || p[0] != "art" || !validID.MatchString(p[1]) || !artKind(p[2]) {
		http.NotFound(w, r)
		return
	}
	c, e := readConfig(p[1])
	if e != nil {
		http.NotFound(w, r)
		return
	}
	path := artworkPath(c, p[1], p[2])
	if path == "" {
		http.NotFound(w, r)
		return
	}
	f, e := os.Open(path)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	var head [512]byte
	n, _ := f.Read(head[:])
	mime := http.DetectContentType(head[:n])
	if !strings.HasPrefix(mime, "image/") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "no-cache")
	stat, e := f.Stat()
	if e != nil {
		return
	}
	f.Seek(0, 0)
	http.ServeContent(w, r, "art", stat.ModTime(), f)
}
func (s *Studio) Library() []Entry {
	out := []Entry{}
	for _, id := range listGames() {
		c, e := readConfig(id)
		row := Entry{ID: id, Config: c}
		if e != nil {
			row.Problem = e.Error()
		}
		out = append(out, row)
	}
	return out
}
func (s *Studio) SaveGame(id string, c gameConfig, create bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validID.MatchString(id) || !filepath.IsLocal(id) {
		return errors.New("ID inválido")
	}
	if create {
		if err := os.Mkdir(gameDir(id), 0755); err != nil {
			return fmt.Errorf("não foi possível criar este ID: %w", err)
		}
	} else if _, err := readConfig(id); err != nil {
		return err
	}
	err := writeConfig(id, c)
	if err != nil && create {
		os.Remove(gameDir(id))
	}
	return err
}
func (s *Studio) Pick(kind string) (string, error) {
	filters := map[string]string{"exe": "*.exe;*.lnk", "core": "*.dll", "rom": "*.*", "art": "*.png;*.jpg;*.jpeg;*.bmp"}
	pat, ok := filters[kind]
	if !ok {
		return "", errors.New("tipo inválido")
	}
	return wr.OpenFileDialog(s.ctx, wr.OpenDialogOptions{Title: "Selecionar arquivo", Filters: []wr.FileFilter{{DisplayName: "Arquivos compatíveis", Pattern: pat}}})
}
func (s *Studio) SetArt(id, k, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, e := readConfig(id); e != nil {
		return e
	}
	if !artKind(k) {
		return errors.New("imagem inválida")
	}
	dst := filepath.Join(gameDir(id), k+".png")
	if k == "cartridge" {
		return fitTransparentPNG(path, dst, 1536, 1024)
	}
	if k == "grid" {
		return cropToPNG(path, dst, 600, 900)
	}
	return cropToPNG(path, dst, 1920, 620)
}
func (s *Studio) CopyGame(id, dest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, e := readConfig(id); e != nil {
		return e
	}
	return copyNew(gameDir(id), dest)
}
func copyNew(src, dest string) error {
	if !validID.MatchString(dest) || !filepath.IsLocal(dest) {
		return errors.New("ID inválido")
	}
	games := filepath.Join(rootDir, "Games")
	rel, e := filepath.Rel(src, games)
	if e == nil && (rel == "." || filepath.IsLocal(rel)) {
		return errors.New("a pasta de origem contém a biblioteca de destino")
	}
	tmp, e := os.MkdirTemp(games, ".import-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(tmp)
	if _, e = os.Lstat(gameDir(dest)); !os.IsNotExist(e) {
		return errors.New("ID já existe ou está inacessível")
	}
	if e = duplicateDir(src, tmp); e != nil {
		return e
	}
	return os.Rename(tmp, gameDir(dest))
}
func (s *Studio) Archive(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, e := readConfig(id); e != nil {
		return e
	}
	dir := filepath.Join(rootDir, "Games-archive")
	if e := os.MkdirAll(dir, 0755); e != nil {
		return e
	}
	return os.Rename(gameDir(id), filepath.Join(dir, id+"-"+time.Now().Format("20060102-150405.000000000")))
}
func (s *Studio) Import() (int, error) {
	dir, e := wr.OpenDirectoryDialog(s.ctx, wr.OpenDialogOptions{Title: "Selecione a pasta Games antiga"})
	if e != nil || dir == "" {
		return 0, e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, e := os.ReadDir(dir)
	if e != nil {
		return 0, e
	}
	n := 0
	for _, ent := range entries {
		if !ent.IsDir() || !validID.MatchString(ent.Name()) {
			continue
		}
		src := filepath.Join(dir, ent.Name())
		if _, e = os.Stat(filepath.Join(src, "config.json")); e != nil {
			continue
		}
		if _, e = os.Stat(gameDir(ent.Name())); e == nil {
			continue
		}
		if e = copyNew(src, ent.Name()); e != nil {
			return n, e
		}
		n++
	}
	return n, nil
}
func (s *Studio) Command(id string) (string, error) {
	if _, e := readConfig(id); e != nil {
		return "", e
	}
	exe, e := os.Executable()
	return fmt.Sprintf("\"%s\" --game \"%s\"", exe, id), e
}
func (s *Studio) Run(id, mode string) error {
	if _, e := readConfig(id); e != nil {
		return e
	}
	if mode != "--game" && mode != "--preview" && mode != "--test-loading" {
		return errors.New("ação inválida")
	}
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	cmd := exec.Command(exe, mode, id)
	if e = cmd.Start(); e != nil {
		return e
	}
	return cmd.Process.Release()
}
func (s *Studio) Prefs() settings.Config { s.mu.Lock(); defer s.mu.Unlock(); return appPrefs }
func (s *Studio) SavePrefs(c settings.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := settings.Save(appSettingsPath(), c); e != nil {
		return e
	}
	loadAppPrefs()
	loadOverlayTheme()
	return nil
}
func (s *Studio) Layout(id string) (ll.Layout, error) {
	if _, e := readConfig(id); e != nil {
		return ll.Layout{}, e
	}
	return loadEffectiveLoadingLayout(id)
}
func (s *Studio) Factory() ll.Layout { return ll.Default() }
func (s *Studio) SaveLayout(id, scope string, l ll.Layout) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, e := readConfig(id); e != nil {
		return e
	}
	switch scope {
	case "game":
		return ll.Save(loadingLayoutFile(id), l)
	case "all":
		_, e := applyDefaultLoadingLayout(l)
		return e
	case "future":
		if e := backupOldLoadingLayout(globalLoadingLayoutFile()); e != nil {
			return e
		}
		return ll.Save(globalLoadingLayoutFile(), l)
	}
	return errors.New("destino inválido")
}
func (s *Studio) Frame(id string, l ll.Layout) (string, error) {
    return s.FrameProgress(id,l,50)
}
func (s *Studio) FrameProgress(id string,l ll.Layout,percent int) (string,error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, e := readConfig(id)
	if e != nil {
		return "", e
	}
	hero := artworkPath(c, id, "hero")
	var bg *img
	if appPrefs.BlurBackground {
		bg = loadBlurry(hero)
	} else {
		bg = loadImg(hero)
	}
	if bg != nil {
		defer bg.close()
	}
	cart := loadImg(artworkPath(c, id, "cartridge"))
	if cart == nil {
		cart = loadImg(artworkPath(c, id, "grid"))
	}
	if cart != nil {
		defer cart.close()
	}
	return renderFramePercent(c, l.Normalize(), cart, bg, heroaccent.FromFile(hero), true,percent)
}
func renderFrame(c gameConfig, l ll.Layout, cart, bg *img, accent uint32, paintBackground bool) (string, error) {
    return renderFramePercent(c,l,cart,bg,accent,paintBackground,50)
}
func renderFramePercent(c gameConfig,l ll.Layout,cart,bg *img,accent uint32,paintBackground bool,percent int) (string,error) {
    if percent<0{percent=0};if percent>100{percent=100}
	// Prévia Full HD: os antigos 960x540 deixavam os textos embaçados ao ampliar
	// o editor ou utilizar Windows com escala de tela alta.
	p, b, e := newBitmapGraphics(1920, 1080)
	if e != nil {
		return "", e
	}
	defer call(pGdipDisposeImage, b)
	drawOverlayPass(p, c, 0, 0, 1920, 1080, percent, "Preparando a tela do jogo", "", cart, bg, accent, 0, l, false, paintBackground)
	p.close()
	dir, e := os.MkdirTemp("", "voyager-preview-")
	if e != nil {
		return "", e
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "frame.png")
	if e = savePNG(b, file); e != nil {
		return "", e
	}
	data, e := os.ReadFile(file)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), e
}
func (s *Studio) CheckUpdate() (string, error) {
  status,err:=probeUpdateStatus()
  if err!=nil{return "",err}
  if status.Available{return status.Latest,nil}
  return "",nil
}
func (s *Studio) Version() string              { return version }
func (s *Studio) CoverPath(id string) (string, error) {
	c, e := readConfig(id)
	if e != nil {
		return "", e
	}
	return artworkPath(c, id, "grid"), nil
}
func (s *Studio) RemoveArt(id, k string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, e := readConfig(id)
	if e != nil {
		return e
	}
	if !artKind(k) {
		return errors.New("imagem inválida")
	}
	p := artworkPath(c, id, k)
	if p == "" {
		return errors.New("imagem ausente")
	}
	if !strings.EqualFold(filepath.Dir(p), gameDir(id)) {
		return errors.New("a imagem está fora da pasta do jogo")
	}
	return os.Rename(p, p+".backup-"+time.Now().Format("20060102-150405.000000000"))
}
func (s *Studio) Diagnose() []string {
	lines := []string{}
	for _, g := range s.Library() {
		if g.Problem != "" {
			lines = append(lines, g.ID+": "+g.Problem)
			continue
		}
		c := g.Config
		paths := []string{c.ExecutablePath}
		if c.Mode == "retroarch" {
			paths = []string{c.RetroarchPath, c.CorePath, c.RomPath}
		} else if paths[0] == "" {
			paths[0] = c.ShortcutPath
		}
		for _, p := range paths {
			if _, e := os.Stat(resolveGameFile(p, gameDir(g.ID))); e != nil {
				lines = append(lines, g.ID+": arquivo ausente — "+p)
			}
		}
	}
	if b, e := os.ReadFile(filepath.Join(rootDir, "VoyagerUpdater-error.log")); e == nil {
		lines = append(lines, "Último erro do atualizador: "+string(b))
	}
	if b, e := os.ReadFile(filepath.Join(rootDir, "VoyagerLaunch-error.log")); e == nil {
		lines = append(lines, "Última falha de execução: "+string(b))
	}
	return lines
}
func (s *Studio) PickArt() (map[string]string, error) {
	p, e := s.Pick("art")
	if e != nil || p == "" {
		return nil, e
	}
	dir, e := os.MkdirTemp("", "voyager-art-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(dir)
	dst := filepath.Join(dir, "preview.png")
	if e = fitTransparentPNG(p, dst, 450, 300); e != nil {
		return nil, e
	}
	b, e := os.ReadFile(dst)
	if e != nil {
		return nil, e
	}
	return map[string]string{"path": p, "preview": "data:image/png;base64," + base64.StdEncoding.EncodeToString(b)}, nil
}

// Build the colorized transparent sprite for WebView2's smooth CSS animation.
func loadingSpritePreview(asset *img,accent uint32,percent int)(string,error){
    if asset==nil{return "",nil}
    f:=math.Min(1,1000.0/float64(asset.W))
    w:=int(math.Max(1,math.Round(float64(asset.W)*f)))
    h:=int(math.Max(1,math.Round(float64(asset.H)*f)))
    p,b,e:=newBitmapGraphics(w,h)
    if e!=nil{return "",e}
    defer call(pGdipDisposeImage,b)
    p.clear(0)
    p.drawLoadingAsset(asset,0,0,w,h,percent,accent,false)
    p.close()
    dir,e:=os.MkdirTemp("","voyager-loading-art-")
    if e!=nil{return "",e}
    defer os.RemoveAll(dir)
    file:=filepath.Join(dir,"sprite.png")
    if e=savePNG(b,file);e!=nil{return "",e}
    raw,e:=os.ReadFile(file)
    if e!=nil{return "",e}
    return "data:image/png;base64,"+base64.StdEncoding.EncodeToString(raw),nil
}

// Only layout or artwork changes require rendering. WebView2 can then move the
// transparent cartridge between the two cached native layers at display rate.
func (s *Studio) AnimationLayers(id string, l ll.Layout) (map[string]string, error) {
    return s.AnimationLayersProgress(id,l,50)
}
func (s *Studio) AnimationLayersProgress(id string,l ll.Layout,percent int) (map[string]string,error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, e := readConfig(id)
	if e != nil {
		return nil, e
	}
	l = l.Normalize()
	hero := artworkPath(c, id, "hero")
	var bg *img
	if appPrefs.BlurBackground {
		bg = loadBlurry(hero)
	} else {
		bg = loadImg(hero)
	}
	if bg != nil {
		defer bg.close()
	}
	accent := heroaccent.FromFile(hero)
	for i, n := range l.Elements {
		if n.ID != ll.Cartridge {
			continue
		}
		base, e := renderFramePercent(c, ll.Layout{Elements: l.Elements[:i]}, nil, bg, accent, true,percent)
		if e != nil {
			return nil, e
		}
		above, e := renderFramePercent(c, ll.Layout{Elements: l.Elements[i+1:]}, nil, nil, accent, false,percent)
		if e != nil {
			return nil, e
		}
		kind := "cartridge"
		if artworkPath(c, id, kind) == "" {
			kind = "grid"
		}
		sprite:=""
        if artwork:=loadImg(artworkPath(c,id,kind));artwork!=nil {
            sprite,e=loadingSpritePreview(artwork,accent,percent)
            artwork.close()
            if e!=nil{return nil,e}
        }
        return map[string]string{"base":base,"above":above,"kind":kind,
          "sprite":sprite,"accent":fmt.Sprintf("#%06X",accent&0xFFFFFF)},nil
	}
	return nil, errors.New("layout sem camada de cartucho")
}

func (s *Studio) UpdateHistory() (string, error) {
	f, e := os.Open(filepath.Join(rootDir, "VoyagerUpdater-history.log"))
	if errors.Is(e, os.ErrNotExist) {
		return "Nenhuma atualização registrada.", nil
	}
	if e != nil {
		return "", e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return "", e
	}
	const max = 8192
	if info.Size() > max {
		if _, e = f.Seek(-max, io.SeekEnd); e != nil {
			return "", e
		}
	}
	b, e := io.ReadAll(io.LimitReader(f, max))
	if e != nil {
		return "", e
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) > 12 {
		lines = lines[len(lines)-12:]
	}
	return strings.Join(lines, "\n"), nil
}
