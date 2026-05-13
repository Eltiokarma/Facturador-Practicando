# Facturador Self-Hosted Perú

Facturador electrónico self-hosted multi-RUC para contribuyentes peruanos. Emite CPE (facturas y boletas) directamente al SEE del Contribuyente de SUNAT, sin pagar mensualidades a facturadores comerciales.

**Estado actual:** MVP funcional con UI completa, cola asíncrona y catálogos. Falta validación end-to-end contra SUNAT beta con certificado real. Ver `CLAUDE.md` para contexto y `docs/manual.pdf` para un manual didáctico.

## Filosofía

- **Self-hosted.** Vos corrés el software, vos tenés tu certificado, vos sos soberano.
- **Cero custodia.** El `.p12` y su passphrase nunca salen de tu servidor.
- **Multi-RUC.** Una sola instancia puede manejar varias empresas (la UI todavía es single-tenant; la DB sí soporta multi).
- **Software libre.** Licencia por definir, dirección clara: open source.

## No es para

- PRICOS con ingresos > 300 UIT que tengan que usar OSE obligatoriamente.
- Quien necesite GRE (Guía de Remisión Electrónica) hoy. GRE llega en v2 antes de julio 2026.

## Componentes

| Carpeta | Qué hace | Stack |
|---------|----------|-------|
| `apps/api` | API REST + worker de cola, auth JWT, persistencia | Go + chi + pgx + asynq |
| `apps/motor` | UBL 2.1 + XAdES-BES + SOAP a SUNAT + CDR | PHP + Greenter |
| `apps/web` | UI completa: emisión, catálogos, listado, PDF | React + TypeScript + Vite + Tailwind + sonner |
| `packages/codigos-sunat` | Catálogos SUNAT (por completar) | JSON |
| `packages/ubl-schemas` | XSDs UBL 2.1 (por completar) | XML |

## Levantar en local

### Instalación con doble click (recomendado para usuarios no técnicos)

Doble click en el script que corresponda a tu sistema:

| SO | Script |
|---|---|
| Windows | `scripts/arrancar-windows.bat` |
| Mac | `scripts/arrancar-mac.command` |
| Linux | `scripts/arrancar-linux.sh` |

La primera vez tarda 3-5 minutos; las siguientes, 30 segundos. Al final abre Chrome con la app y un usuario demo (`demo@local` / `demodemo`). Después podés **"Instalar aplicación"** desde Chrome para tener un ícono en el escritorio. Guía completa en `docs/instalar-en-pc.md`.

### Explorar sin certificado (modo DEMO)

Para conocer el sistema sin necesidad de conseguir certificado SUNAT ni nada más:

```bash
cp .env.example .env
# editar .env: solo POSTGRES_PASSWORD, API_JWT_SECRET y MASTER_KEY
# (los tres se generan con:  openssl rand -hex 32)

docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d --build

docker compose exec api /app/seed-user \
    -email=tu@email -password=demodemo -nombre="Tu nombre" -rol=dueno

# abrir http://localhost:5173 — banner amarillo DEMO arriba
```

Las emisiones se simulan localmente. Los PDFs salen con marca de agua DEMO. Guía completa: `docs/explorar-modo-demo.md`.

### Producción / homologación real

Lo mismo de arriba, y después en la UI:
1. Configuración → completar razón social, dirección, ubigeo.
2. Configuración → subir el `.p12` y su passphrase.
3. Configuración → cargar usuario y clave SOL del usuario secundario.
4. Configuración → destildar "Modo DEMO" y guardar.

Detalles paso a paso en `docs/instalacion.md`. Manual didáctico (30 pág.) en `docs/manual.pdf`.

## Roadmap

- [x] Decisión de stack del motor (ADR 0001).
- [x] Scaffolding del monorepo.
- [x] Docker Compose con postgres + redis + api + motor + web.
- [x] Primer "hola mundo": emitir 1 factura contra SUNAT beta y guardar el CDR.
- [x] Flujo factura + boleta end-to-end.
- [x] PDF imprimible con QR del hash CPE.
- [x] Auth JWT con login, refresh y multi-usuario (roles dueño / contador / cajero).
- [x] Persistencia Postgres con migraciones embebidas y correlativos atómicos.
- [x] UI completa estilo Stripe-like: dashboard, emitir, listado, detalle, clientes, productos.
- [x] Autocomplete de clientes y productos en la pantalla de emisión.
- [x] Cola asíncrona Redis con worker, reintentos exponenciales, polling en la UI.
- [x] Resumen diario consolidado de boletas (RC) con flujo ticket → consulta.
- [x] Configuración del tenant desde la UI (datos del emisor, modo SUNAT, credenciales SOL cifradas AES-256-GCM).
- [x] Multi-tenant real con switch en la UI: pertenencias `user_tenants`, selector de empresa en el sidebar, alta de empresas nuevas, switch-tenant que rota tokens JWT.
- [x] Subir certificado .p12 desde la UI por tenant (un cert por RUC, passphrase cifrada con AES-256-GCM en DB, recarga automática al reiniciar el container).
- [x] Notas de crédito y notas de débito, con motivo SUNAT (catálogos 09/10), referencia al comprobante original, ítems prellenados.
- [x] Modo DEMO por tenant: emisiones simuladas localmente sin tocar SUNAT, banner permanente en la UI, watermark "DEMO" en los PDFs. Permite explorar el sistema sin tener cert ni credenciales reales.
- [x] Detección de caídas de SUNAT: ping periódico a los endpoints, banner global en la UI cuando está down, botón "Reintentar ahora" en comprobantes en estado error.
- [x] Guía de Remisión Electrónica (GRE — tipo 09): modelo completo con destinatario, puntos de partida/llegada, transportista, peso, ítems. Validaciones SUNAT (ubigeo, placa, modalidad pública/privada). OAuth2 real contra api-cpe.sunat.gob.pe vía Greenter\Api; flow ticket → consulta async.
- [x] Comunicación de baja (anulación): anular factura/boleta/nota aceptada por SUNAT dentro de los 7 días. Flow ticket. Marca el comprobante como anulado cuando SUNAT acepta.
- [x] Importación masiva desde CSV: clientes y productos. Validación fila por fila, plantilla descargable, resumen de errores.
- [x] App móvil Android (Capacitor) con impresión Bluetooth térmica ESC/POS (incluye QR del CPE), splash screen, status bar y haptics. La PWA web sigue funcionando con Web Bluetooth API en Chrome.
- [ ] Invitar usuarios a tu tenant (compartir acceso con contador / cajero).
- [ ] Offline real con cola en IndexedDB (hoy la cola es server-side).
- [ ] App iOS (`npx cap add ios`).
- [ ] PWA offline-first con cola local en IndexedDB.
- [ ] GRE (obligatoria desde julio 2026).
- [ ] Notificaciones por email al cliente con el PDF y XML adjuntos.
- [ ] Importación masiva desde Excel / CSV.
- [ ] Distribución como binarios nativos (sin Docker).

## Endpoints API (resumen)

| Método | Path | Auth | Qué hace |
|---|---|---|---|
| GET  | `/health` | — | liveness + modo SUNAT |
| GET  | `/ready` | — | chequea motor PHP |
| POST | `/api/v1/auth/login` | — | login con email + password |
| POST | `/api/v1/auth/refresh` | — | renueva access token |
| GET  | `/api/v1/me` | JWT | datos del usuario logueado |
| POST | `/api/v1/facturas` \| `/api/v1/boletas` | JWT | encola un comprobante (202) |
| GET  | `/api/v1/comprobantes` | JWT | listar con filtros |
| GET  | `/api/v1/comprobantes/{id}` | JWT | detalle |
| GET  | `/api/v1/comprobantes/{id}/pdf` | JWT | representación impresa |
| GET  | `/api/v1/comprobantes/{id}/xml` | JWT | UBL firmado |
| GET  | `/api/v1/comprobantes/{id}/cdr` | JWT | CDR de SUNAT |
| GET/POST/PUT/DELETE | `/api/v1/clientes[/:id]` | JWT | catálogo de clientes |
| GET/POST/PUT/DELETE | `/api/v1/productos[/:id]` | JWT | catálogo de productos |
| GET  | `/api/v1/resumenes` | JWT | listar resúmenes diarios |
| GET  | `/api/v1/resumenes/pendientes?fecha=...` | JWT | boletas pendientes de resumir |
| POST | `/api/v1/resumenes` | JWT | crear y encolar resumen del día |
| GET  | `/api/v1/resumenes/{id}` | JWT | detalle del resumen con sus boletas |
| GET  | `/api/v1/tenant` | JWT | datos del tenant actual |
| PUT  | `/api/v1/tenant` | JWT dueño | actualizar perfil |
| PUT  | `/api/v1/tenant/credenciales` | JWT dueño | rotar usuario/clave SOL |
| POST | `/api/v1/tenant/cert` (multipart) | JWT dueño | subir .p12 + passphrase |
| POST | `/api/v1/tenants` | JWT | crear empresa nueva (te volvés dueño) |
| GET  | `/api/v1/me/tenants` | JWT | empresas del usuario |
| POST | `/api/v1/auth/switch-tenant` | JWT | cambiar de empresa (emite nuevos tokens) |
| POST | `/api/v1/guias` | JWT | emitir guía de remisión (tipo 09) |
| POST | `/api/v1/clientes/importar` (multipart) | JWT | importar clientes desde CSV |
| POST | `/api/v1/productos/importar` (multipart) | JWT | importar productos desde CSV |
| POST | `/api/v1/comprobantes/{id}/anular` | JWT | anular un CPE aceptado (≤7 días) |
| GET  | `/api/v1/anulaciones` | JWT | historial de anulaciones |
| GET  | `/api/v1/anulaciones/{id}` | JWT | detalle de una anulación |
| PUT  | `/api/v1/tenant/gre-credenciales` | JWT dueño | guardar credenciales API GRE (OAuth2) |

## Arquitectura de emisión

```
   [Frontend React]
        │ POST /api/v1/facturas
        ▼
   [API Go]
      ├── valida + recalcula totales server-side
      ├── asigna correlativo atómico (Postgres)
      ├── inserta comprobante con estado=pendiente
      ├── encola job en Redis (asynq)
      └── responde 202 { id, estado: "pendiente" }
        │
        │ (en paralelo, en el mismo proceso)
        ▼
   [Worker Go]
      ├── lee job de Redis
      ├── marca estado=enviando
      ├── llama POST http://motor:8000/emitir
      │       │
      │       ▼
      │   [Motor PHP/Greenter]
      │     ├── construye UBL 2.1
      │     ├── firma XAdES-BES con .p12
      │     ├── envuelve en SOAP + WS-Security
      │     ├── envía a SUNAT (beta o prod)
      │     └── parsea CDR
      │       │
      │       ◄── XML firmado, CDR.zip, hash, estado
      ├── persiste resultado en Postgres
      └── persiste XML/CDR en filesystem (/app/data)

   [Frontend React]
        │ GET /api/v1/comprobantes/{id} (polling cada 2s)
        ▼ hasta que estado deje de ser pendiente/enviando/error
   ✓ toast con el resultado, botones de PDF/XML/CDR habilitados
```

Si SUNAT está caído, asynq reintenta con backoff exponencial (2s, 4s, 8s, … hasta 7 intentos en 48h).
