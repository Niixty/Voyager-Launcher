@echo off
setlocal
chcp 65001 >nul
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\Mover-App.ps1"
set "resultado=%ERRORLEVEL%"
echo.
pause
exit /b %resultado%
