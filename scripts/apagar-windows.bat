@echo off
REM Apaga el Facturador. Los datos se mantienen guardados.

setlocal
cd /d "%~dp0\.."

cls
echo ===============================================
echo   Facturador - apagando...
echo ===============================================
echo.

docker compose down

echo.
echo OK Facturador apagado.
echo.
echo   Tus datos siguen guardados. La proxima vez
echo   que uses "arrancar-windows.bat" van a estar
echo   ahi esperandote.
echo.
echo   Podes cerrar esta ventana.
echo.
pause
