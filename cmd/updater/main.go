//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// The updater is a separate native Windows GUI executable because Windows
// keeps the running launcher's image mapped until its process exits.
func main() {
	if len(os.Args) != 5 || os.Args[1] != "--apply" {
		return
	}
	target, err := filepath.Abs(os.Args[2])
	if err != nil {
		return
	}
	staged, err := filepath.Abs(os.Args[3])
	if err != nil {
		return
	}
	if !strings.EqualFold(filepath.Base(target), "VoyagerLauncher.exe") || !strings.EqualFold(filepath.Base(staged), "VoyagerLauncher.next.exe") || !strings.EqualFold(filepath.Dir(target), filepath.Dir(staged)) {
		return
	}
	pid, err := strconv.Atoi(os.Args[4])
	if err != nil || pid <= 0 {
		return
	}
	// Do not replace the executable while the manager or game is using it.
	kernel := syscall.NewLazyDLL("kernel32.dll")
	open := kernel.NewProc("OpenProcess")
	wait := kernel.NewProc("WaitForSingleObject")
	closeHandle := kernel.NewProc("CloseHandle")
	process, _, _ := open.Call(0x00100000, 0, uintptr(pid))
	if process != 0 {
		wait.Call(process, 30000)
		closeHandle.Call(process)
	} else {
		time.Sleep(1500 * time.Millisecond)
	}
	backup := target + ".bak"
	var finalErr error
	for attempt := 0; attempt < 30; attempt++ {
		_ = os.Remove(backup)
		if err = os.Rename(target, backup); err != nil {
			finalErr = err
			time.Sleep(time.Second)
			continue
		}
		if err = os.Rename(staged, target); err == nil {
			finalErr = nil
			break
		}
		_ = os.Rename(backup, target)
		finalErr = err
		time.Sleep(time.Second)
	}
	if finalErr != nil {
		_ = os.WriteFile(filepath.Join(filepath.Dir(target), "VoyagerUpdater-error.log"), []byte(fmt.Sprintf("%s: %v\n", time.Now().Format(time.RFC3339), finalErr)), 0644)
	}
}
