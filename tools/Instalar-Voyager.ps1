# Primeira instalacao: busca a ultima versao publica gerada pelo GitHub Actions.
# Nao modifica Games, capas, configuracoes, arquivos pessoais nem o repositorio Git.
[CmdletBinding()]
param(
    [string]$Destino
)
$ErrorActionPreference = 'Stop'
if (-not $Destino) {
    $padrao = Join-Path $env:LOCALAPPDATA 'VoyagerLauncher'
    Write-Host "Pasta onde o Voyager sera instalado. Se ja existe, informe a MESMA pasta para preservar o comando do Sunshine."
    $inputDir = Read-Host "Pasta do Voyager [Enter: $padrao]"
    if ([string]::IsNullOrWhiteSpace($inputDir)) { $Destino = $padrao } else { $Destino = $inputDir.Trim('"') }
}
$Destino = [System.IO.Path]::GetFullPath([Environment]::ExpandEnvironmentVariables($Destino))
$api = 'https://api.github.com/repos/Niixty/Voyager-Launcher/releases/latest'
$base = 'https://github.com/Niixty/Voyager-Launcher/releases/download/'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$headers = @{'User-Agent'='Voyager-Windows-Installer'; 'Accept'='application/vnd.github+json'}
Write-Host 'Consultando a ultima versao publicada no GitHub...'
try { $release = Invoke-RestMethod -Uri $api -Headers $headers -TimeoutSec 35 }
catch { throw 'Ainda nao existe uma versao publicada ou a rede falhou. Confirme que o GitHub Actions terminou na aba Actions do repositorio.' }
if ($release.draft -or $release.prerelease -or -not $release.tag_name) { throw 'Release atual nao e estavel.' }
$nomes = @('VoyagerLauncher.exe', 'VoyagerUpdater.exe')
$temp = Join-Path ([IO.Path]::GetTempPath()) ('VoyagerInstall-' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $temp -Force | Out-Null
try {
    foreach ($nome in $nomes) {
        $asset = @($release.assets | Where-Object { $_.name -eq $nome }) | Select-Object -First 1
        if (-not $asset -or -not $asset.browser_download_url -or $asset.digest -notmatch '^sha256:[a-fA-F0-9]{64}$') {
            throw "Release sem $nome validado por SHA-256. Nenhum arquivo instalado."
        }
        $assetURL = [string]$asset.browser_download_url
        if (-not $assetURL.StartsWith($base, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Origem inesperada para $nome. Nenhum arquivo instalado."
        }
        $destTemp = Join-Path $temp $nome
        Write-Host "Baixando $nome ($($release.tag_name))..."
        Invoke-WebRequest -Uri $asset.browser_download_url -Headers $headers -OutFile $destTemp -TimeoutSec 90 -UseBasicParsing
        $hash = (Get-FileHash -LiteralPath $destTemp -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($hash -ne ([string]$asset.digest).Substring(7).ToLowerInvariant()) { throw "SHA-256 invalido para $nome. Nenhum arquivo instalado." }
    }
    New-Item -ItemType Directory -Path $Destino -Force | Out-Null
    # Salva copia dos executaveis antigos antes de substituir; nunca toca em Games.
    foreach ($nome in $nomes) {
        $destExe = Join-Path $Destino $nome
        if (Test-Path -LiteralPath $destExe) {
            Copy-Item -LiteralPath $destExe -Destination ($destExe + '.backup') -Force
        }
    }
    $copiados = @()
    try {
        foreach ($nome in $nomes) {
            Copy-Item -LiteralPath (Join-Path $temp $nome) -Destination (Join-Path $Destino $nome) -Force
            $copiados += $nome
        }
    } catch {
        foreach ($nome in $copiados) {
            $anterior = Join-Path $Destino ($nome + '.backup')
            if (Test-Path -LiteralPath $anterior) {
                Copy-Item -LiteralPath $anterior -Destination (Join-Path $Destino $nome) -Force -ErrorAction SilentlyContinue
            }
        }
        throw "Nao foi possivel instalar todos os arquivos. Feche o Voyager e tente de novo. Detalhes: $($_.Exception.Message)"
    }
    Write-Host "Instalacao concluida: $Destino" -ForegroundColor Green
    Write-Host "Versao: $($release.tag_name)" -ForegroundColor Green
    Write-Host 'A pasta Games, se ja existir, permanece intocada.'
    $agendar = Join-Path $PSScriptRoot 'Ativar-Sincronizacao.ps1'
    & $agendar -PastaVoyager $Destino
} finally {
    Remove-Item -LiteralPath $temp -Force -Recurse -ErrorAction SilentlyContinue
}
