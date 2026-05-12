#!/usr/bin/env bash
# Crea un tenant de prueba para homologación SUNAT.
# RUC genérico de pruebas: 20000000001 (NO usar en producción).
#
# TODO: implementar cuando la API tenga endpoint /tenants.

set -euo pipefail

API="${API:-http://localhost:8080}"

echo "Verificando que la API esté arriba en ${API}…"
curl -sf "${API}/health" >/dev/null || { echo "API no responde"; exit 1; }

echo "TODO: crear tenant de prueba vía POST /api/v1/tenants (aún no implementado)."
