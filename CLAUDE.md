# CLAUDE.md — Facturador Self-Hosted Perú

> Contexto del proyecto para Claude Code. Leer antes de cualquier cambio.

## 1. Qué estamos construyendo

Un **facturador electrónico self-hosted multi-RUC** para contribuyentes peruanos que quieran emitir CPE (Comprobantes de Pago Electrónicos) directamente al SEE del Contribuyente de SUNAT, sin pagar mensualidades a Nubefact / Kesito / Efact / Rapifact / Facpro / etc.

**Modelo de distribución:** software self-hosted. El usuario corre la aplicación en su propia infraestructura (servidor propio, VPS, o incluso laptop). Nosotros NO custodiamos certificados ajenos. Cada instalación es soberana.

**No es un PSE/OSE.** No firmamos en nombre de terceros. Distribuimos software, no servicios de facturación.

## 2. Decisiones ya tomadas

| Tema | Decisión |
|------|----------|
| Distribución | Solo Docker Compose por ahora (binarios nativos en v2+) |
| GRE (Guía de Remisión Electrónica) | NO en MVP. Solo facturas, boletas, notas de crédito y notas de débito. GRE en v2 (obligatoria desde julio 2026, hay que tenerla en el roadmap). |
| Stack motor de facturación (UBL + firma + SOAP) | **Opción B: Go API + microservicio PHP/Greenter** — ver `docs/adr/0001-stack-motor.md` |
| Multi-tenant | Sí, una instancia puede manejar múltiples RUCs (un dueño con 2 empresas, o un contador con varios clientes en su propio servidor) |
| Modo SUNAT | Beta (homologación) y Producción. Toggle por tenant. |

## 3. Endpoints SUNAT (referencia)

- **Beta (homologación):** `https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService`
- **Producción facturas/boletas/notas:** `https://e-factura.sunat.gob.pe/ol-ti-itcpfegem/billService`
- **Producción guías (v2):** `https://e-guiaremision.sunat.gob.pe/ol-ti-itemision-guia-gem/billService`
- **Consulta CDR:** `https://e-factura.sunat.gob.pe/ol-it-wsconscpegem/billConsultService`

Protocolo: SOAP 1.1 con WS-Security (UsernameToken + PasswordText, sin cifrado del password en el header — SUNAT lo recibe en plano sobre HTTPS).

## 4. Stack del motor de facturación: DECIDIDO

**Opción B — Go API + microservicio PHP/Greenter.**

Razón: Greenter resuelve UBL 2.1 + XAdES-BES + SOAP + PDF para Perú, mantenido y probado en producción. Implementar XAdES sobre UBL 2.1 en Go puro es 3-4x más laburo y deuda técnica innecesaria. El motor PHP queda enjaulado, sin acceso a red salvo SUNAT. Ver ADR completo en `docs/adr/0001-stack-motor.md`.

## 5. Stack del resto (no negociable)

- **Frontend:** React PWA (TypeScript + Vite + TailwindCSS). Offline-first con Service Worker.
- **Backend API pública:** Go (chi router). Es el que habla con el frontend.
- **DB:** PostgreSQL. Multi-tenant por columna `tenant_id` (simple, suficiente para el volumen esperado; schema-per-tenant queda como optimización futura si un tenant crece mucho).
- **Auth:** JWT con refresh tokens. Multi-usuario por tenant (dueño + cajeros + contador).
- **Almacenamiento de certificados:** el `.p12` del usuario se sube al servidor del usuario, se encripta en reposo con una passphrase que el dueño introduce al iniciar el container (env var o vault local). NUNCA se guarda la passphrase en DB. Si el container se reinicia, hay que re-introducirla. Esto es a propósito: cero custodia, máxima soberanía.
- **PDF:** representación gráfica bonita. NO el PDF feo del SFS. Template configurable con logo, datos, QR con hash CPE.
- **Cola:** envío asíncrono a SUNAT (los WS de SUNAT son lentos y a veces caen). Redis + worker Go (asynq o river).

## 6. Estructura del monorepo

```
facturador/
├── docker-compose.yml
├── docker-compose.beta.yml
├── CLAUDE.md
├── README.md
├── docs/
│   ├── adr/                    # Architecture Decision Records
│   ├── instalacion.md
│   ├── obtener-cdt.md          # Guía para usuarios finales
│   └── crear-usuario-secundario.md
├── apps/
│   ├── api/                    # Backend Go
│   ├── web/                    # Frontend React PWA
│   └── motor/                  # Motor de facturación (PHP/Greenter)
├── packages/
│   ├── ubl-schemas/            # XSDs de UBL 2.1 SUNAT
│   └── codigos-sunat/          # Catálogos (tipos doc, monedas, unidades, etc.)
└── scripts/
    └── seed-beta.sh            # Datos de prueba para homologación
```

## 7. Flujo de emisión (happy path)

1. Frontend manda POST `/api/v1/facturas` con JSON del comprobante.
2. Backend Go valida estructura, calcula totales (NUNCA confiar en totales del cliente, recalcular IGV server-side), genera correlativo atómico por serie.
3. Backend Go encola job de emisión.
4. Worker desencola, llama al motor de facturación con el JSON.
5. Motor genera UBL 2.1 XML, lo firma con el `.p12` del tenant, lo envuelve en SOAP, lo manda a SUNAT.
6. SUNAT responde con CDR (zip con XML de respuesta). Worker guarda XML enviado + CDR + estado.
7. Worker genera PDF bonito con QR (hash del CPE).
8. Frontend hace polling o WebSocket para ver el estado final.

## 8. Restricciones legales / regulatorias (al 2026)

- UIT 2026: S/ 5,500. Sanción por no emisión: 50% UIT = S/ 2,750 (con 90% rebaja si subsanás).
- Plazo de envío del CPE a SUNAT: 3 días calendario contados desde el día siguiente a la fecha de emisión. Después de eso, SUNAT rechaza aunque el comprobante ya esté en manos del cliente.
- Todo CPE y su CDR debe conservarse mínimo 5 años (responsabilidad del usuario, no nuestra — pero el software debe facilitar respaldos).
- GRE obligatoria desde julio 2026 → roadmap v2.
- OSE obligatorio para PRICOS con ingresos > 300 UIT (S/ 1,650,000 anuales). Nuestro target es MYPE y medianas, no PRICOS — pero documentar que el software NO sirve para PRICOS grandes que requieran OSE.

## 9. Lo que el usuario tiene que conseguir (documentar en `docs/`)

1. **Certificado Digital Tributario (CDT)** — gratis desde Clave SOL → Empresas → Comprobantes de Pago → CDT. Vigencia 3 años. Archivo `.p12`.
2. **Usuario secundario Clave SOL** con permiso de "Emisión de Comprobantes Electrónicos". Sin esto, SUNAT devuelve error 0111.
3. **RUC activo y habido**, afecto a renta de tercera categoría (Régimen General, RMT, RER).

## 10. Identidad del proyecto

- Nombre tentativo: TBD (el usuario decide cuando vea el MVP funcionando).
- Licencia: TBD. Sugerencia: AGPL-3.0 (impide que alguien lo tome, lo cierre y lo venda como SaaS sin contribuir de vuelta) o MIT si queremos máxima adopción.
- Idioma: español peruano. Interfaz en español. Documentación en español.

## 11. Anti-patrones a evitar

- ❌ Hardcodear endpoints, usar config por tenant.
- ❌ Confiar en totales del frontend. Recalcular IGV, ISC, ICBPER server-side siempre.
- ❌ Guardar passphrases de `.p12` en DB.
- ❌ Enviar comprobantes en sincrónico bloqueando la respuesta HTTP al frontend. Cola siempre.
- ❌ Generar correlativos no-atómicos (race conditions = SUNAT te rechaza por duplicado).
- ❌ Asumir que SUNAT está siempre arriba. Reintentar con backoff exponencial.
- ❌ Olvidar el código de detracción, percepción, retención cuando aplique.
- ❌ PDFs feos. El usuario quiere algo que el cliente reciba y diga "qué bonito".

## 12. Próximos pasos para Claude Code

1. ✅ Leído CLAUDE.md.
2. ✅ Decisión del stack del motor (Opción B) escrita en `docs/adr/0001-stack-motor.md`.
3. ✅ Estructura del monorepo levantada.
4. ✅ `docker-compose.yml` con postgres + redis + api + motor + web.
5. ✅ Primer milestone end-to-end: API Go ⇄ motor PHP/Greenter ⇄ SUNAT.
6. ✅ Auth JWT con multi-usuario, roles, refresh tokens, bcrypt.
7. ✅ Persistencia Postgres con migraciones embebidas y correlativos atómicos.
8. ✅ UI completa estilo Stripe-like: dashboard, emitir, listado, detalle, clientes, productos.
9. ✅ PDF imprimible A4 con QR del hash CPE.
10. ✅ Catálogos de clientes y productos con autocomplete en la emisión.
11. ✅ Cola asíncrona Redis + asynq con worker en el mismo proceso, polling en la UI.
12. ⏳ Notas de crédito y débito (próximo grande).
13. ⏳ Resumen diario de boletas (lo exige SUNAT).
14. ⏳ Configuración del tenant desde la UI (subir cert, editar datos).
15. ⏳ Multi-tenant real desde la UI (hoy es single-tenant por env).
16. ⏳ GRE (obligatoria julio 2026).

## 13. Sobre el dueño del proyecto

Gerson, chemical engineer, dueño de restaurante en Juliaca (3825 msnm). Construye este facturador porque está cansado de pagar mensualidades a facturadores comerciales y porque cree (con razón) que muchos otros peruanos están en la misma. Stack que ya maneja: Go, React PWA, PostgreSQL, SQLite. INTJ, prioriza claridad técnica y honestidad por sobre pulido superficial. Hablale en español peruano informal, sin pelos en la lengua. Si una decisión técnica es mala, decíselo. Si algo no se puede, decí por qué.
