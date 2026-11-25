//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	runtime.LockOSThread()
	exe, err := os.Executable()
	if err != nil {
		exitLog(err)
		return
	}
	rootDir = filepath.Dir(exe)
	loadAppPrefs()
	loadOverlayTheme()
	// Verificacao silenciosa acionada pelo Agendador de Tarefas do Windows.
	// Ela nao abre janela, nao bloqueia o Sunshine e nao altera a pasta Games.
	args := os.Args[1:]
	if len(args) == 1 && strings.EqualFold(args[0], "--check-updates") {
		if _, err := probeUpdateStatus(); err != nil {
            logSilentUpdateError(err)
        }
		return
	}
	// Awareness context PER_MONITOR_AWARE_V2; falls back when not supported.
	call(u32.NewProc("SetProcessDpiAwarenessContext"), ^uintptr(3))
	if err := initGraphics(); err != nil {
		exitLog(err)
		return
	}
	defer stopGraphics()
	if len(args) >= 2 && (strings.EqualFold(args[0], "--game") || strings.EqualFold(args[0], "-game")) {
		id := args[1]
		c, e := readConfig(id)
		if e != nil {
			exitLog(fmt.Errorf("configuração %s: %w", id, e))
			return
		}
		o := &overlayState{cfg: c, id: id}
		if e = o.run(); e != nil {
			exitLog(e)
		}
		return
	}
	// Open the real fullscreen overlay without launching any game.
	if len(args) >= 2 && args[0] == "--test-loading" {
		id := args[1]
		c, e := readConfig(id)
		if e != nil {
			exitLog(e)
			return
		}
		o := &overlayState{cfg: c, id: id, preview: true, testLoading: true}
		if e = o.run(); e != nil {
			exitLog(e)
		}
		return
	}
	if len(args) >= 2 && args[0] == "--preview" {
		id := args[1]
		c, e := readConfig(id)
		if e != nil {
			exitLog(e)
			return
		}
		path, e := previewScreenshot(c, id, artworkPath(c, id, "grid"), artworkPath(c, id, "hero"), artworkPath(c, id, "cartridge"))
		if e != nil {
			exitLog(e)
		} else {
			messageBox(0, "Print salvo em:\n"+path, "Voyager Launcher")
		}
		return
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "Games"), 0755); err != nil {
		exitLog(err)
		return
	}
	if len(args) == 0 {
        // Repair legacy scheduled Git sync tasks silently on user startup;
        // never spawn a visible CMD/PowerShell window over a game.
        go migrateSilentScheduledTask()
		if err := newStudio(); err == nil {
			return
		} else {
			logSilentUpdateError(err)
		}
	}
	if err := newManager(); err != nil {
		exitLog(err)
	}
}
