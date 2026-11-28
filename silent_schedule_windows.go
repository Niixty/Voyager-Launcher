//go:build windows

package main

import (
    _ "embed"
    "encoding/base64"
    "encoding/binary"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "syscall"
    "unicode/utf16"
)

//go:embed silent_schedule_repair.ps1
var silentScheduleRepairSource string

// PowerShell's -EncodedCommand expects UTF-16LE (not UTF-8). Single-quote
// shell literals so apostrophes/spaces in installation paths remain literal.
func silentRepairCommand(appDirectory string) string {
    quote := func(s string) string {
        return "'" + strings.ReplaceAll(s, "'", "''") + "'"
    }
    script := strings.ReplaceAll(silentScheduleRepairSource,
        "__VOYAGER_LAUNCHER_DIR__", quote(appDirectory))
    script = strings.ReplaceAll(script, "__VOYAGER_SYNC_WRAPPER__",
        quote(filepath.Join(filepath.Dir(appDirectory), "tools", "Sincronizar-Git-Silencioso.vbs")))
    units := utf16.Encode([]rune(script))
    data := make([]byte, len(units)*2)
    for i, unit := range units {
        binary.LittleEndian.PutUint16(data[i*2:], unit)
    }
    return base64.StdEncoding.EncodeToString(data)
}

// Verify and repair BOTH existing Task Scheduler actions without touching
// triggers, repetition intervals, task enabled state or game data.
//
// The previous implementation wrote a one-time "sync-task-hidden-v2" marker,
// so recreated/legacy tasks could silently keep launching cmd.exe forever.
// We instead inspect actions on each GUI launch and scheduled EXE check.
// The embedded script does nothing for already-correct tasks.
//
// A GUI launcher starts PowerShell with CREATE_NO_WINDOW and never inherits
// a console; WScript runs the Git sync script with its window style hidden.
func migrateSilentScheduledTask() {
    windir := os.Getenv("SystemRoot")
    if windir == "" {
        windir = os.Getenv("WINDIR")
    }
    if windir == "" {
        return
    }
    powershell := filepath.Join(windir, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
    if _, err := os.Stat(powershell); err != nil {
        return
    }
    cmd := exec.Command(powershell,
        "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden",
        "-ExecutionPolicy", "Bypass",
        "-EncodedCommand", silentRepairCommand(rootDir))
    cmd.SysProcAttr = &syscall.SysProcAttr{
        HideWindow: true,
        CreationFlags: 0x08000000, // CREATE_NO_WINDOW
    }
    cmd.Stdout = io.Discard
    cmd.Stderr = io.Discard
    _ = cmd.Run() // Nonfatal: insufficient permissions are noted in local repair log.
}
