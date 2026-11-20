//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
    "os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"voyager/native/internal/updatefiles"
)

// The updater is a separate native Windows GUI executable because Windows
// keeps the running launcher's image mapped until its process exits.
func main() {
	if (len(os.Args) != 5 && len(os.Args) != 6) || os.Args[1] != "--apply" {
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
	restart:=len(os.Args)==6 && os.Args[5]=="--restart"
    if len(os.Args)==6 && !restart {return}
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
		result, _, _ := wait.Call(process, 30000)
		closeHandle.Call(process)
		if result != 0 {
			logError(target, errors.New("launcher ainda em execução"))
			return
		}
	} else {
		time.Sleep(1500 * time.Millisecond)
	}
	unlock, err := updatefiles.Lock(filepath.Dir(target))
	if err != nil {
		logError(target, err)
		return
	}
	defer unlock()
	if err = apply(target, staged); err != nil {
        logError(target, err)
        return
    }
    if restart {
        // Restore the manager immediately after its validated replacement.
        // The updater is a separate process, because Windows locks live EXEs.
        if _,err=os.Stat(target);err!=nil {logError(target,err);return}
        cmd:=exec.Command(target)
        cmd.Dir=filepath.Dir(target)
        if err=cmd.Start();err!=nil {logError(target,fmt.Errorf("não foi possível reabrir Voyager: %w",err));return}
        _=cmd.Process.Release()
    }
}
func logError(target string, err error) {
	_ = os.WriteFile(filepath.Join(filepath.Dir(target), "VoyagerUpdater-error.log"), []byte(fmt.Sprintf("%s: %v\n", time.Now().Format(time.RFC3339), err)), 0644)
}
func apply(target, staged string) error {
	backup := target + ".bak"
	// Recover installations interrupted by the older two-rename algorithm.
	if _, e := os.Stat(target); errors.Is(e, os.ErrNotExist) {
		if e = updatefiles.Backup(backup, target); e != nil {
			return e
		}
	}
	if _, e := os.Stat(staged); errors.Is(e, os.ErrNotExist) {
		return nil
	} else if e != nil {
		return e
	}
	expected, e := os.ReadFile(staged + ".sha256")
	if e != nil {
		return fmt.Errorf("verificação pendente: %w", e)
	}
	actual, e := updatefiles.Digest(staged)
	if e != nil {
		return e
	}
	if !strings.EqualFold(actual, strings.TrimSpace(string(expected))) {
		return errors.New("arquivo preparado diverge do SHA-256 da release")
	}
	if err := updatefiles.Backup(target, backup); err != nil {
		return err
	}
	// Same-volume replacement keeps the live executable available on failure.
	var err error
	for i := 0; i < 30; i++ {
		if err = os.Rename(staged, target); err == nil {
			os.Remove(staged + ".sha256")
			file, e := os.OpenFile(filepath.Join(filepath.Dir(target), "VoyagerUpdater-history.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if e == nil {
				fmt.Fprintf(file, "%s  Instalado SHA-256 %s\n", time.Now().Format(time.RFC3339), actual)
				file.Close()
			}
			return nil
		}
		if !errors.Is(err, syscall.Errno(32)) && !errors.Is(err, syscall.Errno(5)) {
			break
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("a versão anterior continua instalada: %w", err)
}
