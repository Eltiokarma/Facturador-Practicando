#!/bin/bash
# Apaga el Facturador (los datos quedan guardados en Docker).
set -e
cd "$(dirname "$0")/.."

clear
echo "==============================================="
echo "  Facturador — apagando…"
echo "==============================================="
echo ""

docker compose down

echo ""
echo "✓ Facturador apagado."
echo ""
echo "  Tus datos siguen guardados. La próxima vez"
echo "  que uses 'arrancar-mac.command' van a estar"
echo "  ahí esperándote."
echo ""
echo "  Podés cerrar esta ventana."
echo ""
