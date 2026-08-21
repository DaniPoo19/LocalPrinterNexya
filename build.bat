@echo off
echo ========================================================
echo   Compilando Agente Local Nexya Printer para Windows
echo   Destino: Carpeta "LocalPrinter"
echo   (Modo GUI Nativo - Sin Consola CMD)
echo ========================================================
set GOOS=windows
set GOARCH=386

if not exist "LocalPrinter" mkdir "LocalPrinter"

go build -ldflags="-H windowsgui -s -w" -o ./LocalPrinter/nexya-printer.exe ./cmd/daemon
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Fallo la compilacion.
    pause
    exit /b %ERRORLEVEL%
)

if not exist "LocalPrinter\config.json" (
    if exist "config.json" (
        copy /y "config.json" "LocalPrinter\config.json" >nul
    )
)

echo [OK] Compilacion exitosa en carpeta LocalPrinter:
echo      - LocalPrinter\nexya-printer.exe
echo      - LocalPrinter\config.json
echo ========================================================
