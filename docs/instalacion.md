# Instalación y primera factura beta

Esta guía te lleva de "repo recién clonado" a "SUNAT beta me aceptó mi primera factura". Tardás 15-20 minutos si ya tenés tu `.p12` y tu usuario secundario Clave SOL.

## Requisitos

- Docker Engine 24+ y Docker Compose v2.
- 2 GB de RAM mínimo, 2 cores.
- Conexión a internet (para hablar con SUNAT).
- Tu **Certificado Digital Tributario (.p12)** descargado desde Clave SOL — ver [`obtener-cdt.md`](./obtener-cdt.md).
- Un **usuario secundario Clave SOL** con permiso de Emisión de CPE — ver [`crear-usuario-secundario.md`](./crear-usuario-secundario.md).

Para **homologación** (modo beta) podés usar el usuario clásico `MODDATOS / MODDATOS`. Para producción, sí o sí el usuario secundario tuyo.

## Paso 1 — Clonar y poner tu certificado

```bash
git clone <repo> facturador
cd facturador
mkdir -p certs data
cp /ruta/donde/tengas/tu-cert.p12 certs/cert.p12
chmod 600 certs/cert.p12
```

## Paso 2 — Configurar `.env`

```bash
cp .env.example .env
$EDITOR .env
```

Cambiá como mínimo:

```env
TENANT_RUC=20XXXXXXXXX                      # tu RUC real (11 dígitos)
TENANT_RAZON_SOCIAL=MI EMPRESA SAC          # tal como figura en SUNAT
TENANT_DIRECCION_FISCAL=AV. SIEMPRE VIVA 123
TENANT_UBIGEO=150101                         # tu ubigeo SUNAT (6 dígitos)
TENANT_DEPARTAMENTO=LIMA
TENANT_PROVINCIA=LIMA
TENANT_DISTRITO=LIMA

TENANT_USUARIO_SOL=MODDATOS                  # para beta sirve esto
TENANT_CLAVE_SOL=MODDATOS                    # para beta sirve esto

CERT_PASSPHRASE=<la-passphrase-de-tu-p12>    # NUNCA se persiste

API_JWT_SECRET=<32+ caracteres aleatorios>   # openssl rand -hex 32
POSTGRES_PASSWORD=<algo-largo>
```

Generá secrets aleatorios:

```bash
openssl rand -hex 32   # para API_JWT_SECRET
openssl rand -hex 16   # para POSTGRES_PASSWORD si querés
```

## Paso 3 — Levantar el stack en modo beta

```bash
docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d
```

Esto arranca: postgres, redis, motor (PHP/Greenter), api (Go), web (React).

Verificá:

```bash
curl http://localhost:8080/health
# {"status":"ok","sunat_mode":"beta","time":"..."}

curl http://localhost:8080/ready
# {"motor":"ok"}
```

Si `ready` no da OK, esperá unos segundos más (el motor PHP tarda en arrancar la primera vez) o revisá:

```bash
docker compose logs api motor
```

## Paso 4 — Crear el primer usuario para entrar a la UI

```bash
docker compose exec api /app/seed-user \
    -email=tu@email.com \
    -password=ponete-uno-largo \
    -nombre="Tu Nombre" \
    -rol=dueno
```

Roles posibles: `dueno`, `contador`, `cajero`. Con eso ya podés entrar a
`http://localhost:5173` y loguearte. La interfaz tiene tres pantallas:
**Resumen**, **Emitir comprobante** (formulario con ítems dinámicos) y
**Comprobantes** (listado con filtros + detalle con descarga de XML/CDR).

## Paso 5 — Emitir tu primera factura beta

```bash
./scripts/probar-factura.sh
```

Si todo está bien configurado, vas a ver:

```
🎉 SUNAT aceptó tu primera factura beta.
   Mirá ./data/01-F001-00000001/ — XML firmado, CDR y registro JSON.
```

En `./data/01-F001-00000001/` tenés:

- `registro.json` — resumen del comprobante con estado SUNAT.
- `firmado.xml` — el UBL 2.1 firmado con XAdES-BES que se mandó a SUNAT.
- `cdr.zip` — la Constancia de Recepción que devolvió SUNAT.

Esos tres archivos son la prueba legal de que emitiste y SUNAT recibió.

### Si SUNAT rechaza

Mirá el `codigo` y `mensaje` en la respuesta. Los más comunes:

| Código | Qué significa |
|--------|---------------|
| 0111   | El usuario SOL no tiene permiso de emisión. → revisar [`crear-usuario-secundario.md`](./crear-usuario-secundario.md) |
| 2335   | RUC del emisor no está activo/habido. → revisar tu RUC en SUNAT |
| 2017, 2018 | Datos del receptor inválidos. |
| 3001+  | Validaciones de estructura del UBL. Mirá el detalle. |

## Paso 6 — Reemitir, cambiar correlativo, etc.

El script usa `F001-1` por defecto. Para emitir otra:

```bash
CORRELATIVO=2 ./scripts/probar-factura.sh
SERIE=F002 CORRELATIVO=1 ./scripts/probar-factura.sh
```

## Paso 7 — Pasar a producción

**Solo cuando hayas validado el flujo en beta.** No antes.

1. Apagar: `docker compose down`.
2. En `.env` cambiar:
   - `SUNAT_MODE=prod`
   - `TENANT_USUARIO_SOL` y `TENANT_CLAVE_SOL` a las credenciales del **usuario secundario tuyo** (NO `MODDATOS`).
3. Levantar sin el override de beta: `docker compose up -d`.
4. Re-introducir `CERT_PASSPHRASE` (no se persiste — a propósito).
5. Probar con un correlativo separado de tu numeración real antes de emitir comprobantes a clientes.

## Respaldo

- Volumen `pgdata` de Docker → cuando se use DB (versiones posteriores).
- `./certs/cert.p12` → el archivo cifrado por sí mismo. Guardá copia offline.
- `./data/` → contiene XMLs y CDRs de cada comprobante emitido. Respaldá esto regularmente.

Por qué `./data/` y no DB en el MVP: porque para el primer "hola mundo" lo importante es que SUNAT acepte. La DB con multi-usuario, JWT, series con correlativo atómico y todo lo demás llega en los siguientes commits, sin romper este flujo.
