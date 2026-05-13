#!/bin/bash
set -e
cd "$(dirname "$0")/.."

clear
echo "==============================================="
echo "  Facturador — apagando…"
echo "==============================================="
echo ""

docker compose down

echo ""
echo "✓ Facturador apagado. Los datos se mantienen."
echo ""
read -p "Presioná Enter para cerrar." _
