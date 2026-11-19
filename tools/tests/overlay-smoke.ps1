# Exercise both CLI launch paths with a real, short-lived captioned Windows window.
$ErrorActionPreference = 'Stop'
$dir = Join-Path $env:RUNNER_TEMP ('VoyagerOverlay-' + [guid]::NewGuid())
New-Item -ItemType Directory $dir | Out-Null
$launcher = Join-Path $dir 'VoyagerLauncher.exe'
Copy-Item dist/VoyagerLauncher.exe $launcher
try {
    $game = Join-Path $dir 'FixtureGame.exe'
    $source = @'
using System;
using System.Windows.Forms;
using System.IO;
public class FixtureGame {
    [STAThread] public static void Main() {
        var form = new Form { Text = "Voyager UI fixture", Width = 640, Height = 400 };
        form.Shown += (s,e) => File.WriteAllText(System.IO.Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "ready"), DateTime.UtcNow.Ticks.ToString());
        var timer = new Timer { Interval = 12000 };
        timer.Tick += (s,e) => form.Close(); timer.Start(); Application.Run(form);
    }
}
'@
    $sourcePath = Join-Path $dir 'FixtureGame.cs'
    Set-Content $sourcePath $source -Encoding Ascii
    & "$env:WINDIR\Microsoft.NET\Framework64\v4.0.30319\csc.exe" /nologo /target:winexe "/out:$game" /reference:System.Windows.Forms.dll $sourcePath
    if ($LASTEXITCODE) { throw 'Falha na compilacao do jogo de teste.' }
    $shell = New-Object -ComObject WScript.Shell
    $lnk = Join-Path $dir 'Jogo.lnk'
    $shortcut = $shell.CreateShortcut($lnk)
    $shortcut.TargetPath = $game
    $shortcut.Save()
    @{postOpenSeconds=3} | ConvertTo-Json | Set-Content (Join-Path $dir 'settings.json') -Encoding utf8NoBOM
    foreach ($mode in 'executavel','atalho') {
        $path = if ($mode -eq 'atalho') { $lnk } else { $game }
        $folder = Join-Path $dir "Games\$mode"
        New-Item -ItemType Directory $folder -Force | Out-Null
        @{title='Fixture';mode=$mode;executablePath=$path} | ConvertTo-Json | Set-Content (Join-Path $folder 'config.json') -Encoding utf8NoBOM
        $ready = Join-Path $dir 'ready'
        Remove-Item $ready -ErrorAction SilentlyContinue
        $p = Start-Process $launcher -ArgumentList '--game',$mode -PassThru
        if (-not $p.WaitForExit(16000)) { $p.Kill(); throw "Overlay travado para $mode" }
        if (-not (Test-Path $ready)) { throw "Janela de jogo ausente para $mode" }
        $age = ([datetime]::UtcNow.Ticks - [long](Get-Content $ready)) / 10000000
        if ($age -lt 2.6 -or $age -gt 8) { throw "Tempo apos janela: $age s no modo $mode" }
        Write-Host "$mode - janela detectada; overlay encerrado apos $age s"
        Get-Process FixtureGame -ErrorAction SilentlyContinue | Where-Object Path -eq $game | Stop-Process -Force
    }
} finally {
    Remove-Item $dir -Recurse -Force -ErrorAction SilentlyContinue
}
