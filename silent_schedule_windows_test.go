//go:build windows

package main

import (
    "encoding/base64"
    "encoding/binary"
    "strings"
    "testing"
    "unicode/utf16"
)

func decodeSilentRepairCommand(t *testing.T, encoded string) string {
    t.Helper()
    raw, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil { t.Fatal(err) }
    if len(raw)%2 != 0 { t.Fatal("PowerShell command is not UTF-16LE") }
    units := make([]uint16, len(raw)/2)
    for i := range units { units[i] = binary.LittleEndian.Uint16(raw[i*2:]) }
    return string(utf16.Decode(units))
}

func TestSilentTaskRepairIsEmbeddedAndHandlesQuotedPaths(t *testing.T) {
    script := decodeSilentRepairCommand(t, silentRepairCommand(`C:\Users\O'Brien\Voyager Launcher\app`))
    if strings.Contains(script, "__VOYAGER_") {
        t.Fatal("PowerShell path template was not fully replaced")
    }
    for _, expected := range []string{
        `'C:\Users\O''Brien\Voyager Launcher\app'`,
        `'C:\Users\O''Brien\Voyager Launcher\tools\Sincronizar-Git-Silencioso.vbs'`,
        "Voyager-Git-Sync",
        "Voyager-EXE-Update",
        "New-ScheduledTaskAction",
        "Set-ScheduledTask",
        "Get-ScheduledTask",
        "--check-updates",
        "wscript.exe",
    } {
        if !strings.Contains(script, expected) {
            t.Fatalf("missing embedded task-repair behavior: %s", expected)
        }
    }
    if strings.Contains(script, "Register-ScheduledTask") || strings.Contains(script, "Unregister-ScheduledTask") {
        t.Fatal("repair must never create or delete scheduled tasks")
    }
}

func TestSilentTaskRepairNoOneShotMarker(t *testing.T) {
    if strings.Contains(silentScheduleRepairSource, "sync-task-hidden-v2") {
        t.Fatal("legacy one-time migration marker prevents repairing recreated tasks")
    }
}
