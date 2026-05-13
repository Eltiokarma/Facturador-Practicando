@echo off
REM Doble click sobre este archivo levanta el Facturador y abre Chrome.
REM La primera vez tarda 3-5 minutos compilando. Podes cerrar la ventana
REM cuando termine.

setlocal
cd /d "%~dp0\.."

cls
echo ===============================================
echo   Facturador - arrancando...
echo ===============================================
echo.

REM 1. Verificar Docker
docker info >nul 2>&1
if errorlevel 1 (
    echo X Docker no esta corriendo.
    echo.
    echo   Abri Docker Desktop ^(el icono de la ballena
    echo   en la barra de tareas^) y espera a que diga
    echo   "Engine running". Despues volve a tocar este
    echo   archivo.
    echo.
    pause
    exit /b 1
)
echo OK Docker corriendo

REM 2. Crear .env si no existe
if not exist .env (
    echo.
    echo  ^> Primera vez: generando archivo de configuracion...
    copy /Y .env.example .env >nul
    echo. >> .env
    echo # === valores generados automaticamente === >> .env
    for /f "delims=" %%i in ('openssl rand -hex 16') do echo POSTGRES_PASSWORD=%%i >> .env
    for /f "delims=" %%i in ('openssl rand -hex 32') do echo API_JWT_SECRET=%%i >> .env
    for /f "delims=" %%i in ('openssl rand -hex 32') do echo MASTER_KEY=%%i >> .env
    echo OK .env creado
)

REM 3. Levantar el stack
echo.
echo  ^> Levantando servicios ^(la primera vez tarda unos minutos^)...
docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d --build
if errorlevel 1 (
    echo.
    echo X Falló el arranque. Revisa los mensajes arriba.
    pause
    exit /b 1
)

REM 4. Esperar a que el frontend responda
echo.
echo  ^> Esperando que el frontend este listo...
set /a tries=0
:wait
set /a tries+=1
if %tries% gtr 90 (
    echo X El frontend no respondio. Revisa los logs con: docker compose logs web
    pause
    exit /b 1
)
timeout /t 1 /nobreak >nul
curl -sf http://localhost:5173 >nul 2>&1
if errorlevel 1 goto wait
echo OK Frontend respondiendo

REM 5. Crear primer usuario si la DB esta vacia
echo.
echo  ^> Verificando primer usuario...
for /f "delims=" %%i in ('docker compose exec -T postgres psql -U facturador -d facturador -tAc "SELECT COUNT(*) FROM users;" 2^>nul') do set USER_COUNT=%%i
if "%USER_COUNT%"=="0" (
    echo   Creando usuario demo: demo@local / demodemo
    docker compose exec -T api /app/seed-user -email=demo@local -password=demodemo -nombre=Demo -rol=dueno
)
if "%USER_COUNT%"=="" (
    echo   Creando usuario demo: demo@local / demodemo
    docker compose exec -T api /app/seed-user -email=demo@local -password=demodemo -nombre=Demo -rol=dueno
)

REM 6. Abrir Chrome
echo.
echo  ^> Abriendo Facturador en Chrome...
start chrome http://localhost:5173 2>nul || start http://localhost:5173

echo.
echo ===============================================
echo   OK Facturador listo en http://localhost:5173
echo ===============================================
echo.
echo   Usuario:    demo@local
echo   Contrasena: demodemo
echo.
echo   Esta ventana se puede cerrar.
echo   Para apagar el sistema, doble click en
echo   "apagar-windows.bat".
echo.
pause
