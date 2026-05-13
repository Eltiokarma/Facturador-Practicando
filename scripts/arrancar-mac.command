#!/bin/bash
# Doble click sobre este archivo levanta el Facturador y abre Chrome.
# Si es la primera vez, también compila — puede tardar 3-5 minutos.
# Cuando termina, podés cerrar esta ventana.

set -e
cd "$(dirname "$0")/.."

clear
echo "==============================================="
echo "  Facturador — arrancando…"
echo "==============================================="
echo ""

# 1. Verificar que Docker esté corriendo
if ! docker info >/dev/null 2>&1; then
    echo "✗ Docker no está corriendo."
    echo ""
    echo "  Abrí Docker Desktop (el ícono de la ballena en"
    echo "  la barra superior) y esperá a que diga 'Engine"
    echo "  running'. Después volvé a tocar este archivo."
    echo ""
    osascript -e 'display dialog "Docker no está corriendo. Abrí Docker Desktop y esperá a que esté listo." with title "Facturador" buttons {"OK"} default button 1 with icon caution'
    exit 1
fi
echo "✓ Docker corriendo"

# 2. Crear .env si no existe (primera vez)
if [ ! -f .env ]; then
    echo ""
    echo "→ Primera vez: generando archivo de configuración…"
    cp .env.example .env
    {
        echo ""
        echo "# === valores generados automáticamente ==="
        echo "POSTGRES_PASSWORD=$(openssl rand -hex 16)"
        echo "API_JWT_SECRET=$(openssl rand -hex 32)"
        echo "MASTER_KEY=$(openssl rand -hex 32)"
    } >> .env
    echo "✓ .env creado"
fi

# 3. Levantar el stack
echo ""
echo "→ Levantando servicios (la primera vez tarda unos minutos)…"
docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d --build

# 4. Esperar a que el frontend responda
echo ""
echo "→ Esperando que el frontend esté listo…"
for i in $(seq 1 90); do
    if curl -sf http://localhost:5173 >/dev/null 2>&1; then
        echo "✓ Frontend respondiendo"
        break
    fi
    sleep 1
done

# 5. Verificar/crear primer usuario si la DB está vacía
echo ""
echo "→ Verificando primer usuario…"
USER_COUNT=$(docker compose exec -T postgres psql -U facturador -d facturador -tAc "SELECT COUNT(*) FROM users;" 2>/dev/null | tr -d '[:space:]' || echo "0")
if [ "$USER_COUNT" = "0" ] || [ -z "$USER_COUNT" ]; then
    echo "  Creando usuario demo: demo@local / demodemo"
    docker compose exec -T api /app/seed-user \
        -email=demo@local -password=demodemo \
        -nombre="Demo" -rol=dueno || true
fi

# 6. Abrir Chrome
echo ""
echo "→ Abriendo Facturador en Chrome…"
open -a "Google Chrome" http://localhost:5173 2>/dev/null || open http://localhost:5173

echo ""
echo "==============================================="
echo "  ✓ Facturador listo en http://localhost:5173"
echo "==============================================="
echo ""
echo "  Usuario:    demo@local"
echo "  Contraseña: demodemo"
echo ""
echo "  Esta ventana se puede cerrar."
echo "  Para apagar el sistema, doble click en"
echo "  'apagar-mac.command'."
echo ""
