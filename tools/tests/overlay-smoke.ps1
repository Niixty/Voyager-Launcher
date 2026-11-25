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
    Add-Type -AssemblyName System.Drawing
    foreach ($mode in 'executavel','atalho') {
        $path = if ($mode -eq 'atalho') { $lnk } else { $game }
        $folder = Join-Path $dir "Games\$mode"
        New-Item -ItemType Directory $folder -Force | Out-Null
        @{title='Fixture';mode=$mode;executablePath=$path} | ConvertTo-Json | Set-Content (Join-Path $folder 'config.json') -Encoding utf8NoBOM
        # A translucent PNG is essential: previous tests exercised no sprite,
        # so the slow alpha-rendering path never ran during the 10 s preview.
        $bmp = New-Object System.Drawing.Bitmap(260,340)
        $gfx = [System.Drawing.Graphics]::FromImage($bmp)
        $gfx.Clear([System.Drawing.Color]::Transparent)
        $brush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255,65,170,220))
        $gfx.FillEllipse($brush,28,30,205,276)
        $brush.Dispose()
        $gfx.Dispose()
        $art = if ($mode -eq 'atalho') { 'cartridge.png' } else { 'grid.png' }
        $bmp.Save((Join-Path $folder $art), [System.Drawing.Imaging.ImageFormat]::Png)
        $bmp.Dispose()
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
    # The ten-second preview must not start any configured executable.
    Remove-Item (Join-Path $dir 'ready') -ErrorAction SilentlyContinue
    $startedAt = Get-Date
    $preview = Start-Process $launcher -ArgumentList '--test-loading','executavel' -PassThru
    if (-not $preview.WaitForExit(14000)) {
        $preview.Kill()
        throw 'Teste visual do loading nao encerrou em 10 segundos.'
    }
    $seconds = ((Get-Date) - $startedAt).TotalSeconds
    if ($seconds -lt 9 -or $seconds -gt 13.5) { throw "Duracao inesperada do teste: $seconds s" }
    if (Test-Path (Join-Path $dir 'ready')) { throw 'O teste de loading abriu o jogo!' }
    Write-Host "Preview do loading encerrado em $seconds s sem iniciar o jogo"
    # The overlay is intentionally WS_EX_NOACTIVATE: ESC must work even when
    # the launcher's UI still has focus. Simulate the native key message.
    Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class VoyagerPreviewTest {
    [DllImport("user32.dll", CharSet=CharSet.Unicode)]
    public static extern IntPtr FindWindow(string className, string title);
    [DllImport("user32.dll")]
    public static extern bool PostMessage(IntPtr hwnd, uint message, IntPtr wParam, IntPtr lParam);
}
'@
    $escPreview = Start-Process $launcher -ArgumentList '--test-loading','atalho' -PassThru
    $hwnd = [IntPtr]::Zero
    for ($i=0; $i -lt 35 -and $hwnd -eq [IntPtr]::Zero; $i++) {
        Start-Sleep -Milliseconds 100
        $hwnd = [VoyagerPreviewTest]::FindWindow('VoyagerLoadingNativeV6', $null)
    }
    if ($hwnd -eq [IntPtr]::Zero) {
        $escPreview.Kill()
        throw 'Janela do teste de loading nao foi criada.'
    }
    if (-not [VoyagerPreviewTest]::PostMessage($hwnd,0x100,[IntPtr]27,[IntPtr]::Zero)) {
        $escPreview.Kill()
        throw 'Nao foi possivel enviar ESC ao overlay.'
    }
    if (-not $escPreview.WaitForExit(4500)) {
        $escPreview.Kill()
        throw 'Teste de loading ignorou ESC e nao encerrou imediatamente.'
    }
    Write-Host 'Preview com cartucho encerrado via ESC sem travamento'
} finally {
    Remove-Item $dir -Recurse -Force -ErrorAction SilentlyContinue
}
