# Embedded in the Windows GUI executable: repair existing Voyager scheduled tasks.
# Never create tasks, alter their triggers, or open a console.
$ErrorActionPreference = 'Stop'
$launcherDir = __VOYAGER_LAUNCHER_DIR__
$syncWrapper = __VOYAGER_SYNC_WRAPPER__
$launcherExe = Join-Path $launcherDir 'VoyagerLauncher.exe'
$repoDir = Split-Path -Parent $launcherDir
$windowsDir = $env:SystemRoot
if ([string]::IsNullOrWhiteSpace($windowsDir)) { $windowsDir = $env:WINDIR }
$wscriptExe = Join-Path $windowsDir 'System32\wscript.exe'
$logPath = Join-Path $env:LOCALAPPDATA 'VoyagerLauncher\scheduled-task-repair.log'

function Write-RepairLog([string]$message) {
    try {
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $logPath) -ErrorAction Stop | Out-Null
        Add-Content -LiteralPath $logPath -Encoding UTF8 -Value ((Get-Date -Format 'yyyy-MM-dd HH:mm:ss') + ' ' + $message) -ErrorAction Stop
    } catch { }
}

function Repair-VoyagerTask([string]$name, [string]$exe, [string]$arguments, [string]$workingDirectory) {
    try {
        $task = Get-ScheduledTask -TaskPath '\' -TaskName $name -ErrorAction SilentlyContinue
        if ($null -eq $task) { return }
        $actions = @($task.Actions)
        $same = $actions.Count -eq 1
        if ($same) {
            $oldExe = [Environment]::ExpandEnvironmentVariables([string]$actions[0].Execute).Trim('"')
            $same = [string]::Equals($oldExe, $exe, [StringComparison]::OrdinalIgnoreCase) -and
                [string]::Equals(([string]$actions[0].Arguments).Trim(), $arguments, [StringComparison]::Ordinal) -and
                [string]::Equals([Environment]::ExpandEnvironmentVariables([string]$actions[0].WorkingDirectory), $workingDirectory, [StringComparison]::OrdinalIgnoreCase)
        }
        if ($same) { return } # Preserve existing triggers, no operation if correct.
        $action = New-ScheduledTaskAction -Execute $exe -Argument $arguments -WorkingDirectory $workingDirectory
        Set-ScheduledTask -TaskPath '\' -TaskName $name -Action $action -ErrorAction Stop | Out-Null
        Write-RepairLog "OK: $name updated to a silent executable; existing triggers preserved."
    } catch {
        Write-RepairLog ("WARN: Could not repair " + $name + ': ' + $_.Exception.Message)
    }
}

try {
    Import-Module ScheduledTasks -ErrorAction Stop
    if ((Test-Path -LiteralPath $syncWrapper -PathType Leaf) -and
        (Test-Path -LiteralPath $wscriptExe -PathType Leaf)) {
        # WScript.exe is a GUI host and the VBS runs PowerShell fully hidden.
        $gitArgs = '//B //Nologo "' + $syncWrapper + '"'
        Repair-VoyagerTask 'Voyager-Git-Sync' $wscriptExe $gitArgs $repoDir
    } else {
        # Do not replace a working task with an unavailable sync script.
        $gitTask = Get-ScheduledTask -TaskPath '\' -TaskName 'Voyager-Git-Sync' -ErrorAction SilentlyContinue
        if ($null -ne $gitTask) {
            $gitActions = @($gitTask.Actions)
            if ($gitActions.Count -ne 1 -or
                ([string]$gitActions[0].Execute) -match '(?i)(cmd|powershell|cscript)(\.exe)?$') {
                Write-RepairLog 'WARN: Git sync wrapper missing; run ATIVAR-SINCRONIZACAO.cmd in the current clone.'
            }
        }
    }
    if (Test-Path -LiteralPath $launcherExe -PathType Leaf) {
        # Launcher is built with the Windows GUI subsystem; --check-updates is silent.
        Repair-VoyagerTask 'Voyager-EXE-Update' $launcherExe '--check-updates' $launcherDir
    }
} catch {
    Write-RepairLog ('WARN: Windows task repair unavailable: ' + $_.Exception.Message)
}
