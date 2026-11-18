# Registra tarefas no Agendador do Windows sem privilegios de administrador.
# Requer o clone do repositorio via GitHub Desktop.
[CmdletBinding()]
param(
    [string]$PastaVoyager = (Join-Path $env:LOCALAPPDATA 'VoyagerLauncher')
)
$ErrorActionPreference = 'Stop'
$source = Join-Path $PSScriptRoot 'Sincronizar-Git.ps1'
$source = (Resolve-Path -LiteralPath $source).Path
$launcher = Join-Path $PastaVoyager 'VoyagerLauncher.exe'
if (-not (Test-Path -LiteralPath (Join-Path $PSScriptRoot '..\.git'))) {
    throw 'Primeiro clone/publice este repositorio pelo GitHub Desktop (.git ausente).'
}
if (-not (Test-Path -LiteralPath $launcher -PathType Leaf)) {
    Write-Warning "EXE nao encontrado em: $launcher. A tarefa de atualizar o programa nao sera registrada."
}
function Agendar([string]$nome, [string]$comando, [int]$minutos) {
    $args = @('/Create','/F','/SC','MINUTE','/MO',"$minutos",'/TN',$nome,'/TR',$comando)
    & schtasks.exe @args
    if ($LASTEXITCODE -ne 0) { throw "Falha ao registrar tarefa $nome. Verifique permissao do Agendador." }
}
$syncCommand = 'powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -File "' + $source + '"'
Agendar 'Voyager-Git-Sync' $syncCommand 10
Write-Host 'Fonte: sincronizacao segura a cada 10 minutos (somente se a copia local estiver limpa).' -ForegroundColor Green
if (Test-Path -LiteralPath $launcher -PathType Leaf) {
    $updateCommand = '"' + $launcher + '" --check-updates'
    Agendar 'Voyager-EXE-Update' $updateCommand 30
    Write-Host 'Programa: verificacao de novas versoes a cada 30 minutos.' -ForegroundColor Green
}
Write-Host 'Atualizacoes locais nao sao enviadas sozinhas: faca Commit + Push no GitHub Desktop.'
