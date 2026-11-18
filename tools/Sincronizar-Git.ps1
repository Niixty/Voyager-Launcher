# Sincroniza APENAS alteracoes remotas recebidas da branch main.
# Nunca faz reset --hard, push, commit ou remove mudancas locais.
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$logFile = Join-Path $env:LOCALAPPDATA 'VoyagerLauncher\sync-git.log'
New-Item -ItemType Directory -Path (Split-Path -Parent $logFile) -Force | Out-Null
function Log([string]$msg) {
    $line = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') $msg"
    Add-Content -LiteralPath $logFile -Value $line -Encoding UTF8
    Write-Output $line
}
function BuscarGit {
    $found = Get-Command git.exe -ErrorAction SilentlyContinue
    if ($found) { return $found.Source }
    $candidates = @(
        "$env:ProgramFiles\Git\cmd\git.exe",
        "${env:ProgramFiles(x86)}\Git\cmd\git.exe",
        "$env:LOCALAPPDATA\Programs\Git\cmd\git.exe"
    )
    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate -PathType Leaf) { return $candidate }
    }
    $desktopDir = Join-Path $env:LOCALAPPDATA 'GitHubDesktop'
    if (Test-Path -LiteralPath $desktopDir) {
        $versions = Get-ChildItem -LiteralPath $desktopDir -Directory -Filter 'app-*' | Sort-Object Name -Descending
        foreach ($dir in $versions) {
            $candidate = Join-Path $dir.FullName 'resources\app\git\cmd\git.exe'
            if (Test-Path -LiteralPath $candidate -PathType Leaf) { return $candidate }
        }
    }
    return $null
}
try {
    $git = BuscarGit
    if (-not $git) { Log 'SKIP: Git nao localizado. Instale Git for Windows ou GitHub Desktop.'; exit 0 }
    if (-not (Test-Path -LiteralPath (Join-Path $repo '.git'))) { Log 'SKIP: esta pasta nao e um clone Git.'; exit 0 }
    & $git -C $repo rev-parse --is-inside-work-tree *> $null
    if ($LASTEXITCODE -ne 0) { Log 'SKIP: Git invalido.'; exit 0 }
    $branch = (& $git -C $repo branch --show-current).Trim()
    if ($LASTEXITCODE -ne 0 -or $branch -ne 'main') { Log "SKIP: branch atual '$branch' (somente main)."; exit 0 }
    $changes = @(& $git -C $repo status --porcelain --untracked-files=normal)
    if ($LASTEXITCODE -ne 0) { Log 'ERRO: nao foi possivel ler status.'; exit 1 }
    if ($changes.Count -gt 0) { Log 'SKIP: ha alteracoes locais. Revise no GitHub Desktop antes de sincronizar.'; exit 0 }
    & $git -C $repo fetch origin main --quiet
    if ($LASTEXITCODE -ne 0) { Log 'ERRO: fetch falhou. Verifique rede/login.'; exit 1 }
    $local = (& $git -C $repo rev-parse HEAD).Trim()
    $remote = (& $git -C $repo rev-parse refs/remotes/origin/main).Trim()
    if ($local -eq $remote) { Log 'OK: fonte ja esta atualizada.'; exit 0 }
    & $git -C $repo merge --ff-only origin/main
    if ($LASTEXITCODE -ne 0) { Log 'SKIP: historico divergiu. Resolva pelo GitHub Desktop.'; exit 0 }
    Log "OK: codigo atualizado: $($local.Substring(0,7)) -> $($remote.Substring(0,7))."
} catch {
    Log "ERRO: $($_.Exception.Message)"
    exit 1
}
