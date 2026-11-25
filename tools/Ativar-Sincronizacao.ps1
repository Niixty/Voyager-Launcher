# Registra sincronizacao de fonte e atualizacoes do EXE no Agendador do Windows.
# O agendamento usa as APIs do Windows, sem interpretar caminhos com espacos via schtasks.exe.
# As tarefas usam o usuario atual (somente quando a sessao estiver iniciada).
[CmdletBinding()]
param(
    [string]$PastaVoyager
)
$ErrorActionPreference = 'Stop'

$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$source = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot 'Sincronizar-Git.ps1')).Path
if ([string]::IsNullOrWhiteSpace($PastaVoyager)) { $PastaVoyager = Join-Path $repo 'app' }
$PastaVoyager = [IO.Path]::GetFullPath([Environment]::ExpandEnvironmentVariables($PastaVoyager))
$launcher = Join-Path $PastaVoyager 'VoyagerLauncher.exe'

if (-not (Test-Path -LiteralPath (Join-Path $repo '.git'))) {
    throw 'Primeiro clone/publice este repositorio pelo GitHub Desktop (.git ausente).'
}

try {
    Import-Module ScheduledTasks -ErrorAction Stop
} catch {
    throw "Nao foi possivel carregar o modulo ScheduledTasks do Windows: $($_.Exception.Message)"
}

$usuario = [Security.Principal.WindowsIdentity]::GetCurrent().Name
$principal = New-ScheduledTaskPrincipal -UserId $usuario -LogonType Interactive -RunLevel Limited
$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 5)

function Agendar {
    param(
        [Parameter(Mandatory=$true)][string]$Nome,
        [Parameter(Mandatory=$true)][string]$Executavel,
        [Parameter(Mandatory=$true)][string]$Argumentos,
        [Parameter(Mandatory=$true)][int]$Minutos
    )
    # Separar Execute e Argument evita os problemas de aspas do /TR do schtasks.exe.
    $action = New-ScheduledTaskAction -Execute $Executavel -Argument $Argumentos -WorkingDirectory $repo
    $trigger = New-ScheduledTaskTrigger -Once -At ((Get-Date).AddMinutes(1)) `
        -RepetitionInterval (New-TimeSpan -Minutes $Minutos) `
        -RepetitionDuration (New-TimeSpan -Days 3650)
    try {
        Register-ScheduledTask -TaskName $Nome -Action $action -Trigger $trigger `
            -Principal $principal -Settings $settings -Force -ErrorAction Stop | Out-Null
        $registered = Get-ScheduledTask -TaskName $Nome -ErrorAction Stop
        if ($registered.State -eq 'Disabled') {
            throw 'A tarefa foi criada, mas esta desativada.'
        }
        Write-Host "OK: $Nome (a cada $Minutos minutos)." -ForegroundColor Green
    } catch {
        throw "Falha ao registrar $Nome para o usuario $usuario. Detalhes: $($_.Exception.Message)"
    }
}

# WScript é um host sem janela de console. O antigo powershell.exe podia
# mostrar uma janela por instantes antes de aplicar -WindowStyle Hidden.
$silent = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot 'Sincronizar-Git-Silencioso.vbs')).Path
$wscript = Join-Path $env:WINDIR 'System32\wscript.exe'
$syncArgs = '//B //Nologo "' + $silent + '"'
Agendar -Nome 'Voyager-Git-Sync' -Executavel $wscript -Argumentos $syncArgs -Minutos 10

if (Test-Path -LiteralPath $launcher -PathType Leaf) {
    Agendar -Nome 'Voyager-EXE-Update' -Executavel $launcher -Argumentos '--check-updates' -Minutos 30
} else {
    Write-Warning "EXE nao encontrado em: $launcher. Tarefa Voyager-EXE-Update nao registrada."
    Write-Host 'Execute INSTALAR-VOYAGER.cmd ou informe a pasta correta do aplicativo e tente novamente.'
}

Write-Host 'Fonte: sincronizacao segura de origin/main (somente quando a copia local estiver limpa).'
Write-Host 'Alteracoes locais devem ser enviadas com Commit + Push no GitHub Desktop.'
