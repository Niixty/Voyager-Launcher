# Primeira instalacao: busca a ultima versao publica gerada pelo GitHub Actions.
# Nao modifica Games, capas, configuracoes, arquivos pessoais nem o repositorio Git.
[CmdletBinding()]
param(
    [string]$Destino
)
$ErrorActionPreference = 'Stop'
if (-not $Destino) {
    $repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
    $padrao = Join-Path $repo 'app'
    Write-Host "Instalacao padrao em app/ dentro do repositorio. Para migrar dados existentes, execute MOVER-APP.cmd antes."
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
    $lock = [IO.File]::Open((Join-Path $Destino '.voyager-update.lock'), 'OpenOrCreate', 'ReadWrite', 'None')
    $installed = @()
    try {
        foreach ($nome in $nomes) {
            $target = Join-Path $Destino $nome
            $staged = Join-Path $Destino ([Guid]::NewGuid().ToString('N') + '.tmp')
            try {
                [IO.File]::Copy((Join-Path $temp $nome), $staged)
                $previous = [IO.File]::Exists($target)
                if ($previous) {
                    [IO.File]::Replace($staged, $target, ($target + '.backup'))
                } else {
                    [IO.File]::Move($staged, $target)
                }
                $installed += @{ Path = $target; Previous = $previous }
            } finally {
                [IO.File]::Delete($staged)
            }
        }
    } catch {
        $failure = $_
        [array]::Reverse($installed)
        foreach ($item in $installed) {
            if ($item.Previous) {
                [IO.File]::Replace(($item.Path + '.backup'), $item.Path, $null)
            } else {
                [IO.File]::Delete($item.Path)
            }
        }
        throw "Instalacao interrompida. Feche o Voyager e tente novamente. $failure"
    } finally {
        $lock.Dispose()
    }
    Write-Host "Instalacao concluida: $Destino" -ForegroundColor Green
    Write-Host "Versao: $($release.tag_name)" -ForegroundColor Green
    Write-Host 'A pasta Games, se ja existir, permanece intocada.'
    $agendar = Join-Path $PSScriptRoot 'Ativar-Sincronizacao.ps1'
    & $agendar -PastaVoyager $Destino
} finally {
    Remove-Item -LiteralPath $temp -Force -Recurse -ErrorAction SilentlyContinue
}
