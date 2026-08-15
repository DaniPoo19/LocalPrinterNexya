@echo off
echo ========================================================
echo   Compilando Agente Local Nexya Printer para Windows
echo   (Modo GUI Nativo - Sin Consola CMD)
echo ========================================================
set GOOS=windows
set GOARCH=386
go build -ldflags="-H windowsgui -s -w" -o nexya-printer.exe ./cmd/daemon
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Fallo la compilacion.
    pause
    exit /b %ERRORLEVEL%
)
echo [OK] Compilacion exitosa: nexya-printer.exe (Modo GUI)
echo ========================================================
