#!/usr/bin/env bash
# Emite UNA factura de prueba contra SUNAT beta.
#
# Antes de correr:
#   1) Tu .p12 está en ./certs/cert.p12
#   2) .env tiene TENANT_RUC, TENANT_RAZON_SOCIAL, CERT_PASSPHRASE, etc.
#   3) Levantaste el stack:
#         docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d
#
# El RUC del receptor 20000000001 sirve para pruebas. Si querés probar
# con otro, cambialo en el JSON de abajo.

set -euo pipefail

API="${API:-http://localhost:8080}"
SERIE="${SERIE:-F001}"
CORRELATIVO="${CORRELATIVO:-1}"
FECHA="${FECHA:-$(date +%Y-%m-%d)}"

echo "→ Esperando que la API responda en ${API}…"
for i in $(seq 1 30); do
    if curl -sf "${API}/health" >/dev/null 2>&1; then
        break
    fi
    sleep 1
    if [ "$i" = "30" ]; then
        echo "✗ La API no respondió en 30s. ¿Está levantada?" >&2
        exit 1
    fi
done
echo "✓ API responde."

echo "→ Verificando que el motor esté listo (a través de /ready)…"
ready=$(curl -s "${API}/ready" || true)
echo "  ${ready}"

read -r -d '' BODY <<JSON || true
{
  "tipo": "01",
  "serie": "${SERIE}",
  "correlativo": ${CORRELATIVO},
  "fecha_emision": "${FECHA}",
  "moneda": "PEN",
  "tipo_operacion": "0101",
  "receptor": {
    "tipo_doc": "6",
    "num_doc": "20000000001",
    "razon_social": "CLIENTE DE PRUEBA SAC",
    "direccion": "AV. CLIENTE 456"
  },
  "items": [
    {
      "codigo": "P001",
      "descripcion": "Servicio de consultoría",
      "unidad": "ZZ",
      "cantidad": 1,
      "valor_unitario": 100.00,
      "afectacion_igv": "10"
    }
  ]
}
JSON

echo "→ Enviando POST ${API}/api/v1/facturas (serie ${SERIE}, correlativo ${CORRELATIVO})…"
echo "  Body:"
echo "${BODY}" | sed 's/^/    /'

resp=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d "${BODY}" \
    "${API}/api/v1/facturas")

body=$(echo "${resp}" | sed '$d')
code=$(echo "${resp}" | tail -n1)

echo ""
echo "← HTTP ${code}"
echo "${body}" | (command -v jq >/dev/null && jq . || cat)
echo ""

case "${code}" in
    200)
        echo "🎉 SUNAT aceptó tu primera factura beta."
        echo "   Mirá ./data/01-${SERIE}-$(printf '%08d' ${CORRELATIVO})/ — XML firmado, CDR y registro JSON."
        ;;
    422)
        echo "⚠ SUNAT rechazó. Revisá el código y mensaje arriba (ver Anexo de códigos SUNAT)."
        ;;
    *)
        echo "✗ Error inesperado. Revisá logs:  docker compose logs api motor"
        exit 1
        ;;
esac
