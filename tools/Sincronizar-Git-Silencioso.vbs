Option Explicit
' Runs the existing Git sync script without ever creating a console window.
' WScript.exe is a GUI executable; PowerShell and any Git child inherit a
' hidden window. No elevated privileges or new network permissions required.
Dim fs, shell, script, command, ps
Set fs = CreateObject("Scripting.FileSystemObject")
Set shell = CreateObject("WScript.Shell")
script = fs.BuildPath(fs.GetParentFolderName(WScript.ScriptFullName), "Sincronizar-Git.ps1")
If Not fs.FileExists(script) Then WScript.Quit 0
ps = shell.ExpandEnvironmentStrings("%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe")
If Not fs.FileExists(ps) Then WScript.Quit 0
command = Chr(34) & ps & Chr(34) & " -NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -File " & Chr(34) & script & Chr(34)
shell.Run command, 0, True
