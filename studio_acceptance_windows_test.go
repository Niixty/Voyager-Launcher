//go:build windows && studioacceptance

package main

import (
	"context"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	wr "github.com/wailsapp/wails/v2/pkg/runtime"
)

// The driver is compiled only for acceptance tests. No debugging port or test
// bindings are exposed by the distributed application.
//
//go:embed tools/tests/studio-smoke.js
var studioDriver string

type AcceptanceProbe struct {
	fixture string
	result  chan string
}

// Progress breadcrumbs make WebView2 driver timeouts diagnosable on CI.
func (p *AcceptanceProbe) Progress(stage string) {
    fmt.Printf("STUDIO_ACCEPTANCE_STEP: %s\n",stage)
}
func (p *AcceptanceProbe) Done(problem string) {
	select {
	case p.result <- problem:
	default:
	}
}
func (p *AcceptanceProbe) ChooseFile() error {
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		dlg := call(u32.NewProc("FindWindowW"), uintptr(wide("#32770")), uintptr(wide("Selecionar arquivo")))
		var pid uint32
		if dlg != 0 {
			call(pGetWindowThreadProcessId, dlg, uintptr(unsafe.Pointer(&pid)))
		}
		if pid == uint32(os.Getpid()) {
			edit := uintptr(0)
			cb := syscall.NewCallback(func(h, param uintptr) uintptr {
				var class [80]uint16
				call(u32.NewProc("GetClassNameW"), h, uintptr(unsafe.Pointer(&class[0])), 80)
				if syscall.UTF16ToString(class[:]) != "Edit" || call(pIsWindowVisible, h) == 0 {
					return 1
				}
				// The common dialog nests its filename Edit under control 1148.
				for parent := h; parent != 0 && parent != dlg; parent = call(u32.NewProc("GetParent"), parent) {
					if call(u32.NewProc("GetDlgCtrlID"), parent) == 1148 {
						edit = h
						return 0
					}
				}
				return 1
			})
			call(u32.NewProc("EnumChildWindows"), dlg, cb, 0)
			if edit != 0 {
				var result uintptr
				if call(u32.NewProc("SendMessageTimeoutW"), edit, 0x000c, 0, uintptr(wide(p.fixture)), 2, 3000, uintptr(unsafe.Pointer(&result))) == 0 {
					return fmt.Errorf("campo Nome do arquivo sem resposta")
				}
				accept := call(u32.NewProc("GetDlgItem"), dlg, 1)
				if accept == 0 {
					return fmt.Errorf("botão Abrir ausente")
				}
				call(pPostMessage, accept, 0x00f5, 0, 0)
				return nil
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("campo Nome do arquivo não encontrado no diálogo do processo")

}
func (p *AcceptanceProbe) Capture(name string) error {
	if name != "biblioteca" && name != "editor" {
		return fmt.Errorf("nome inválido")
	}
	if err := os.MkdirAll("evidence", 0755); err != nil {
		return err
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-File", "tools/tests/capture-window.ps1", "-AppProcess", strconv.Itoa(os.Getpid()), "-Destination", filepath.Join("evidence", name+".png"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", out, err)
	}
	return nil
}
func TestStudioAcceptance(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	old := rootDir
	rootDir = t.TempDir()
	defer func() { rootDir = old }()
	if e := os.MkdirAll(filepath.Join(rootDir, "Games"), 0755); e != nil {
		t.Fatal(e)
	}
	loadAppPrefs()
	loadOverlayTheme()
	if e := initGraphics(); e != nil {
		t.Fatal(e)
	}
	defer stopGraphics()
	probe := &AcceptanceProbe{fixture: filepath.Join(rootDir, "fixture.png"), result: make(chan string, 1)}
	im := image.NewNRGBA(image.Rect(0, 0, 96, 64))
	for y := 10; y < 54; y++ {
		for x := 16; x < 80; x++ {
			im.SetNRGBA(x, y, color.NRGBA{R: 42, G: 150, B: 138, A: 255})
		}
	}
	f, e := os.Create(probe.fixture)
	if e != nil {
		t.Fatal(e)
	}
	e = png.Encode(f, im)
	f.Close()
	if e != nil {
		t.Fatal(e)
	}
	assets, e := fs.Sub(web, "desktop")
	if e != nil {
		t.Fatal(e)
	}
	opts := studioOptions(&Studio{}, assets)
	opts.Bind = append(opts.Bind, probe)
	var problem string
	opts.OnDomReady = func(ctx context.Context) {
		go func() {
			wr.WindowExecJS(ctx, studioDriver)
			select {
			case problem = <-probe.result:
			case <-time.After(100 * time.Second):
				problem = "tempo esgotado no WebView2"
			}
			wr.Quit(ctx)
		}()
	}
	if e = wails.Run(opts); e != nil {
		t.Fatal(e)
	}
	if problem != "" {
		t.Fatal(problem)
	}
	for _, file := range []string{configFile("Smoke"), loadingLayoutFile("Smoke"), appSettingsPath()} {
		if _, e = os.Stat(file); e != nil {
			t.Fatal(e)
		}
	}
}
