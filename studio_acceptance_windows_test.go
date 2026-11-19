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
	"testing"
	"time"

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

func (p *AcceptanceProbe) Done(problem string) {
	select {
	case p.result <- problem:
	default:
	}
}
func (p *AcceptanceProbe) ChooseFile() error {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-File", "tools/tests/choose-file.ps1", "-AppProcess", strconv.Itoa(os.Getpid()), "-ImagePath", p.fixture).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", out, err)
	}
	return nil
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
