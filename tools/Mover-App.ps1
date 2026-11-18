# Copia a instalacao antiga para app/ e reconfigura o agendamento.
# A origem fica intacta; nenhum arquivo ja presente no destino e sobrescrito.
[CmdletBinding()]
param([string]$Origem = (Join-Path $env:LOCALAPPDATA 'VoyagerLauncher'))
$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
if (-not (Test-Path -LiteralPath (Join-Path $repo '.git'))) {
    throw 'Repositorio nao encontrado. Use a pasta do clone do GitHub Desktop.'
}
$app = Join-Path $repo 'app'
$Origem = [IO.Path]::GetFullPath([Environment]::ExpandEnvironmentVariables($Origem))
if ([string]::Equals($Origem.TrimEnd('\'), $app.TrimEnd('\'), [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Origem e destino sao iguais. Nenhum arquivo alterado.'
}
$ativos = @(Get-Process -Name 'VoyagerLauncher','VoyagerUpdater' -ErrorAction SilentlyContinue)
if ($ativos.Count -gt 0) {
    throw 'Feche VoyagerLauncher e VoyagerUpdater antes de migrar.'
}
if (-not (Test-Path -LiteralPath $Origem -PathType Container)) {
    throw "Instalacao antiga nao encontrada: $Origem. Use INSTALAR-VOYAGER.cmd para instalar do zero."
}
New-Item -ItemType Directory -Path $app -Force | Out-Null
$copiados = 0
$preservados = 0
function CopiarSemSobrescrever {
    param([string]$De, [string]$Para)
    foreach ($item in (Get-ChildItem -LiteralPath $De -Force -ErrorAction Stop)) {
        if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
            Write-Warning "Link simbolico ignorado: $($item.FullName)"
            continue
        }
        $destino = Join-Path $Para $item.Name
        if ($item.PSIsContainer) {
            if (Test-Path -LiteralPath $destino -PathType Leaf) { throw "Conflito arquivo/pasta: $destino" }
            New-Item -ItemType Directory -Path $destino -Force | Out-Null
            CopiarSemSobrescrever -De $item.FullName -Para $destino
        } elseif (Test-Path -LiteralPath $destino) {
            $script:preservados++
        } else {
            Copy-Item -LiteralPath $item.FullName -Destination $destino -ErrorAction Stop
            $script:copiados++
        }
    }
}
Write-Host "Origem: $Origem"
Write-Host "Destino: $app"
CopiarSemSobrescrever -De $Origem -Para $app
$exe = Join-Path $app 'VoyagerLauncher.exe'
if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) {
    throw 'VoyagerLauncher.exe nao encontrado. Execute INSTALAR-VOYAGER.cmd e instale em app.'
}
Write-Host "Arquivos copiados: $copiados; ja existentes e preservados: $preservados" -ForegroundColor Green
& (Join-Path $PSScriptRoot 'Ativar-Sincronizacao.ps1') -PastaVoyager $app
Write-Host "Concluido. Novo programa: $exe" -ForegroundColor Green
Write-Host 'A instalacao antiga foi preservada como seguranca.'
Write-Host 'Atualize o caminho do executavel na configuracao do Sunshine.'
