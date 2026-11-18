foreach ($name in @('Voyager-Git-Sync','Voyager-EXE-Update')) {
    & schtasks.exe /Delete /F /TN $name 2>$null
}
Write-Host 'Tarefas do Voyager removidas (se existiam).'
