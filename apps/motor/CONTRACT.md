# Contrato HTTP entre `apps/api` (Go) y `apps/motor` (PHP/Greenter)

Este archivo define el contrato estable entre los dos servicios. Si rompés este contrato, hay que versionarlo (`/v2/emitir`).

## Principios

1. El motor es **stateless**. No tiene DB. Si se cae, se reinicia y no perdió nada.
2. El motor solo habla con SUNAT y con quien lo llama (la API). No expone nada al frontend.
3. Toda llamada incluye `tenant_ruc` y, cuando se requiera firma, las credenciales SUNAT y el certificado **descifrados** que el backend Go tiene en memoria.

## Endpoints

### `GET /health`

Liveness check. Devuelve `200` si el proceso está vivo.

### `POST /emitir`

Genera UBL 2.1, firma con XAdES-BES, envía a SUNAT, devuelve CDR.

**Request (JSON):**

```json
{
  "modo": "beta",
  "tenant": {
    "ruc": "20123456789",
    "razon_social": "MI EMPRESA SAC",
    "nombre_comercial": "Mi Empresa",
    "direccion_fiscal": "Av. Siempre Viva 123",
    "ubigeo": "150101",
    "usuario_sol": "MODDATOS",
    "clave_sol": "MODDATOS",
    "cert_pem": "-----BEGIN CERTIFICATE-----...",
    "cert_key_pem": "-----BEGIN PRIVATE KEY-----..."
  },
  "comprobante": {
    "tipo": "01",
    "serie": "F001",
    "correlativo": 1,
    "fecha_emision": "2026-05-12",
    "moneda": "PEN",
    "receptor": {
      "tipo_doc": "6",
      "num_doc": "20987654321",
      "razon_social": "CLIENTE SAC",
      "direccion": "Av. Cliente 456"
    },
    "items": [
      {
        "codigo": "P001",
        "descripcion": "Producto demo",
        "unidad": "NIU",
        "cantidad": 1,
        "valor_unitario": 100.00,
        "precio_unitario": 118.00,
        "afectacion_igv": "10",
        "porcentaje_igv": 18,
        "igv": 18.00,
        "total": 118.00
      }
    ],
    "totales": {
      "gravado": 100.00,
      "exonerado": 0,
      "inafecto": 0,
      "gratuito": 0,
      "igv": 18.00,
      "isc": 0,
      "icbper": 0,
      "total": 118.00
    }
  }
}
```

**Notas:**

- `modo`: `"beta"` o `"prod"`. El motor selecciona el endpoint SUNAT correcto.
- `cert_pem` y `cert_key_pem`: PEM en plano. Llegan descifrados desde el backend Go (que tiene la passphrase en memoria). El motor NO tiene acceso a la passphrase ni al .p12 cifrado.
- `tipo`: catálogo SUNAT 01. `01` factura, `03` boleta, `07` nota crédito, `08` nota débito.
- `afectacion_igv`: catálogo SUNAT 07. `10` = gravado, `20` = exonerado, `30` = inafecto, etc.
- Los totales vienen pre-calculados por la API Go, pero el motor puede validar consistencia.

**Response 200 (éxito):**

```json
{
  "estado": "aceptado",
  "xml_firmado": "<base64>",
  "cdr_zip": "<base64>",
  "hash_cpe": "abc123...",
  "codigo": "0",
  "mensaje": "La Factura numero F001-1, ha sido aceptada"
}
```

**Response 200 (aceptado con observaciones):**

```json
{
  "estado": "aceptado_con_obs",
  "xml_firmado": "<base64>",
  "cdr_zip": "<base64>",
  "hash_cpe": "abc123...",
  "codigo": "0",
  "mensaje": "...",
  "observaciones": ["4000 - ...", "4001 - ..."]
}
```

**Response 200 (rechazado por SUNAT):**

```json
{
  "estado": "rechazado",
  "xml_firmado": "<base64>",
  "codigo": "2335",
  "mensaje": "El RUC del emisor no se encuentra activo"
}
```

**Response 500 (error del motor o de red contra SUNAT):**

```json
{
  "estado": "error",
  "codigo": "MOTOR_TIMEOUT",
  "mensaje": "..."
}
```

### `POST /anular`

(v2) Comunicación de baja para anular comprobantes ya enviados.

## Versionado

- Versión actual: `v1` (implícito en la base URL).
- Si rompemos el contrato, montar `/v2/emitir` y mantener `/v1` durante un release de transición.
