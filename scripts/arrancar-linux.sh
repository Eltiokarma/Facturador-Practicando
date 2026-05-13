#!/bin/bash
# Doble click sobre este archivo levanta el Facturador y abre Chrome.
# (En la mayoría de distros Linux hay que marcarlo "Ejecutable" desde
# las propiedades del archivo la primera vez, o correr:
#     chmod +x arrancar-linux.sh
# )

set -e
cd "$(dirname "$0")/.."

clear
echo "==============================================="
echo "  Facturador — arrancando…"
echo "==============================================="
echo ""

# 1. Verificar Docker
if ! docker info >/dev/null 2>&1; then
    echo "✗ Docker no está corriendo."
    echo ""
    echo "  Arrancalo con:  sudo systemctl start docker"
    echo "  o instalalo desde: docs.docker.com/engine/install"
    echo ""
    read -p "Presioná Enter para cerrar." _
    exit 1
fi
echo "✓ Docker corriendo"

# 2. Crear .env si no existe
if [ ! -f .env ]; then
    echo ""
    echo "→ Primera vez: generando configuración…"
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

# 3. Levantar
echo ""
echo "→ Levantando servicios (primera vez tarda unos minutos)…"
docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d --build

# 4. Esperar
echo ""
echo "→ Esperando que el frontend esté listo…"
for i in $(seq 1 90); do
    if curl -sf http://localhost:5173 >/dev/null 2>&1; then
        echo "✓ Frontend respondiendo"
        break
    fi
    sleep 1
done

# 5. Primer usuario
echo ""
echo "→ Verificando primer usuario…"
USER_COUNT=$(docker compose exec -T postgres psql -U facturador -d facturador -tAc "SELECT COUNT(*) FROM users;" 2>/dev/null | tr -d '[:space:]' || echo "0")
if [ "$USER_COUNT" = "0" ] || [ -z "$USER_COUNT" ]; then
    echo "  Creando usuario demo: demo@local / demodemo"
    docker compose exec -T api /app/seed-user \
        -email=demo@local -password=demodemo \
        -nombre="Demo" -rol=dueno || true
fi

# 6. Abrir Chrome (probamos varios comandos según la distro)
echo ""
echo "→ Abriendo Facturador…"
URL="http://localhost:5173"
if command -v google-chrome >/dev/null 2>&1; then google-chrome "$URL" &
elif command -v chromium >/dev/null 2>&1; then chromium "$URL" &
elif command -v chromium-browser >/dev/null 2>&1; then chromium-browser "$URL" &
elif command -v xdg-open >/dev/null 2>&1; then xdg-open "$URL" &
else echo "  No encontré ningún navegador. Abrí manualmente: $URL"; fi

echo ""
echo "==============================================="
echo "  ✓ Facturador listo en http://localhost:5173"
echo "==============================================="
echo ""
echo "  Usuario:    demo@local"
echo "  Contraseña: demodemo"
echo ""
echo "  Esta ventana se puede cerrar."
echo "  Para apagar: doble click en 'apagar-linux.sh'."
echo ""
