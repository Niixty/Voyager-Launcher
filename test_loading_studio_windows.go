//go:build windows

package main

import (
    "errors"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "sync"
    "time"
    "unsafe"

    ll "voyager/native/internal/loadinglayout"
)

var loadingTestState struct {
    sync.Mutex
    running bool
}

// FindWindowEx iterates all top-level overlay windows. A previous preview
// may still exist, so checking only the first matching class is unreliable.
var findOverlayWindow = u32.NewProc("FindWindowExW")

// TestLoading runs only the independent 10-second preview executable.
// Unlike Run, it waits for the actual window to appear and for the preview
// process to finish. The Wails promise stays pending, so the editor can show
// status and prevent multiple overlapping tests; Esc completes it early.
func (s *Studio) TestLoading(id string, layout ll.Layout) error {
    if _,err:=readConfig(id);err!=nil{return err}
    loadingTestState.Lock()
    if loadingTestState.running {
        loadingTestState.Unlock()
        return errors.New("Já existe um teste de loading aberto. Pressione Esc para encerrar.")
    }
    loadingTestState.running=true
    loadingTestState.Unlock()
    defer func(){
        loadingTestState.Lock()
        loadingTestState.running=false
        loadingTestState.Unlock()
    }()

    dir,err:=os.MkdirTemp("","voyager-test-layout-*")
    if err!=nil{return err}
    defer os.RemoveAll(dir)
    path:=filepath.Join(dir,"layout.json")
    if err=ll.Save(path,layout);err!=nil{return err}
    exe,err:=os.Executable()
    if err!=nil{return err}
    cmd:=exec.Command(exe,"--test-loading",id)
    cmd.Dir=rootDir
    cmd.Env=append(os.Environ(),"VOYAGER_TEST_LAYOUT="+path)
    if err=cmd.Start();err!=nil{return fmt.Errorf("Não foi possível iniciar o teste: %w",err)}
    completed:=make(chan error,1)
    go func(){completed<-cmd.Wait()}()

    // Opening a process successfully is not enough: verify its native
    // fullscreen window actually exists (and belongs to this child PID).
    ready:=false
    timer:=time.NewTimer(6*time.Second)
    defer timer.Stop()
    probe:=time.NewTicker(65*time.Millisecond)
    defer probe.Stop()
    for !ready {
        select{
        case err=<-completed:
            if err!=nil{return fmt.Errorf("O teste terminou antes de abrir a tela: %w",err)}
            return errors.New("O teste terminou antes de mostrar a tela de loading.")
        case <-probe.C:
            var previous uintptr
            for i:=0;i<64;i++ {
                hwnd:=call(findOverlayWindow,0,previous,uintptr(unsafe.Pointer(overlayClassName)),0)
                if hwnd==0 {break}
                var owner uint32
                call(pGetWindowThreadProcessId,hwnd,uintptr(unsafe.Pointer(&owner)))
                if owner==uint32(cmd.Process.Pid) {ready=true;break}
                previous=hwnd
            }
        case <-timer.C:
            _=cmd.Process.Kill()
            <-completed
            return errors.New("A janela do teste não apareceu. O processo foi encerrado com segurança.")
        }
    }

    // Normal return after 10 seconds, or immediately after the user's Esc.
    // This fallback is separate from the child process watchdog.
    select{
    case err=<-completed:
        if err!=nil{return fmt.Errorf("O loading encerrou com erro: %w",err)}
        return nil
    case <-time.After(loadingTestDuration+4*time.Second):
        _=cmd.Process.Kill()
        <-completed
        return errors.New("O teste excedeu o limite de tempo e foi encerrado.")
    }
}
