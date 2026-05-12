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

## Levantar en local (homologación)

```bash
cp .env.example .env
# editar .env: TENANT_RUC, TENANT_RAZON_SOCIAL, CERT_PASSPHRASE, API_JWT_SECRET
mkdir -p certs data
cp /donde/tengas/tu-cert.p12 certs/cert.p12

docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d --build

# crear el primer usuario
docker compose exec api /app/seed-user \
    -email=tu@email.com -password=loquesea123 -nombre="Tu Nombre" -rol=dueno

# abrir http://localhost:5173
```

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
- [ ] Notas de crédito y notas de débito.
- [ ] Subir certificado .p12 desde la UI (hoy: archivo + env, requiere reinicio).
- [ ] Multi-tenant real con onboarding desde la UI.
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
| GET  | `/api/v1/tenant` | JWT | datos del tenant (sin clave SOL) |
| PUT  | `/api/v1/tenant` | JWT dueño | actualizar perfil del tenant |
| PUT  | `/api/v1/tenant/credenciales` | JWT dueño | rotar usuario/clave SOL |

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
