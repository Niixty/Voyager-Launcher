@echo off
chcp 65001 >nul
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\Ativar-Sincronizacao.ps1"
echo.
pause
