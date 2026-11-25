//go:build windows

package main

import (
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "syscall"
)

// On existing Windows installations, a 10-minute Git sync task may still
// launch powershell.exe directly. Migrate it to WScript without showing CMD,
// only when the bundled repository tools are present and the old task exists.
func migrateSilentScheduledTask() {
    repo:=filepath.Dir(rootDir)
    script:=filepath.Join(repo,"tools","Sincronizar-Git-Silencioso.vbs")
    if _,err:=os.Stat(script);err!=nil{return}
    cfg,err:=os.UserCacheDir()
    if err!=nil{return}
    mark:=filepath.Join(cfg,"VoyagerLauncher","sync-task-hidden-v2")
    if _,err=os.Stat(mark);err==nil{return}
    sysroot:=os.Getenv("SystemRoot")
    if sysroot==""{sysroot=os.Getenv("WINDIR")}
    if sysroot==""{return}
    wscript:=filepath.Join(sysroot,"System32","wscript.exe")
    scheduler:=filepath.Join(sysroot,"System32","schtasks.exe")
    if _,err=os.Stat(wscript);err!=nil{return}
    action:=fmt.Sprintf(`"%s" //B //Nologo "%s"`,wscript,script)
    cmd:=exec.Command(scheduler,"/Change","/TN","Voyager-Git-Sync","/TR",action)
    cmd.SysProcAttr=&syscall.SysProcAttr{
        HideWindow:true,
        CreationFlags:0x08000000, // CREATE_NO_WINDOW
    }
    cmd.Stdout=io.Discard
    cmd.Stderr=io.Discard
    if err=cmd.Run();err!=nil{return} // Absent task / insufficient access: no change.
    if err=os.MkdirAll(filepath.Dir(mark),0700);err==nil{
        _=os.WriteFile(mark,[]byte("WScript task enabled\n"),0600)
    }
}
