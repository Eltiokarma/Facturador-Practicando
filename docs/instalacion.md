# Instalación

Esta guía cubre la instalación en un servidor Linux (Debian/Ubuntu, Alpine, etc.) o en una laptop con Docker. Si querés correrlo en Windows, usá WSL2 + Docker Desktop — funciona igual.

## Requisitos

- Docker Engine 24+ y Docker Compose v2.
- 2 GB de RAM mínimo, 2 cores.
- Conexión a internet estable (para hablar con SUNAT).
- Un dominio + certificado TLS si vas a exponerlo a internet (recomendado: poner detrás de un reverse proxy como Caddy o Traefik). Para uso local en una LAN no hace falta.

## Antes de arrancar — lo que tenés que tener listo

1. **Tu RUC** activo y habido, afecto a renta de tercera (Régimen General, RMT o RER).
2. **Certificado Digital Tributario (CDT)** descargado desde Clave SOL como archivo `.p12`. Ver `docs/obtener-cdt.md`.
3. **Usuario secundario Clave SOL** con permiso "Emisión de Comprobantes Electrónicos". Ver `docs/crear-usuario-secundario.md`.

Si te falta cualquiera de los tres, parar y conseguirlo primero. Sin esto el sistema no emite nada.

## Paso 1 — Clonar y configurar

```bash
git clone <repo> facturador
cd facturador
cp .env.example .env
```

Editá `.env`:

- Cambiá `POSTGRES_PASSWORD` por algo largo y aleatorio.
- Cambiá `API_JWT_SECRET` por algo aleatorio de al menos 32 caracteres. Tip:
  ```bash
  openssl rand -hex 32
  ```
- Dejá `SUNAT_MODE=beta` mientras estés probando. Cambialo a `prod` solo cuando hayas validado que todo el flujo funciona en homologación.

## Paso 2 — Levantar en modo beta (homologación)

```bash
docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d
```

Esperá unos segundos y verificá:

```bash
curl http://localhost:8080/health
```

Deberías ver algo así:

```json
{"status":"ok","sunat_mode":"beta","time":"2026-05-12T18:00:00Z"}
```

Y la UI en `http://localhost:5173`.

## Paso 3 — Subir el certificado y crear el primer tenant

(En desarrollo. Ver el milestone "primer hola mundo" en el README.)

## Paso 4 — Pasar a producción

Solo cuando hayas validado el flujo completo en beta:

1. Apagar el stack: `docker compose down`.
2. Cambiar en `.env`: `SUNAT_MODE=prod`.
3. Volver a levantar sin el override de beta: `docker compose up -d`.
4. Re-introducir la passphrase del `.p12` (no se persiste, es a propósito).

## Respaldo

- `pgdata` (volumen de Docker) contiene toda la base de datos.
- `./certs/` contiene los `.p12` cifrados.
- Hacer snapshots regulares de ambos. Idealmente fuera del servidor.

## Por qué no usamos Kubernetes

Porque no hace falta. El target son MYPE y medianas que corren esto en un VPS o una laptop. Docker Compose alcanza y sobra. Si tu empresa necesita Kubernetes, probablemente necesitás un PSE/OSE, no este software.
