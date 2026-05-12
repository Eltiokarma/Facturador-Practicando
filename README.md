# Facturador Self-Hosted Perú

Facturador electrónico self-hosted multi-RUC para contribuyentes peruanos. Emite CPE (facturas, boletas, notas de crédito y notas de débito) directamente al SEE del Contribuyente de SUNAT, sin pagar mensualidades a facturadores comerciales.

**Estado actual:** scaffolding inicial. Ver `CLAUDE.md` para el contexto completo y `docs/adr/0001-stack-motor.md` para la decisión técnica del motor.

## Filosofía

- **Self-hosted.** Vos corrés el software, vos tenés tu certificado, vos sos soberano.
- **Cero custodia.** El `.p12` y su passphrase nunca salen de tu servidor.
- **Multi-RUC.** Una sola instancia maneja varias empresas si las tenés (o si sos contador).
- **Software libre.** Licencia por definir, pero la dirección es claramente open source.

## No es para

- PRICOS con ingresos > 300 UIT que tengan que usar OSE obligatoriamente.
- Quien necesite GRE (Guía de Remisión Electrónica) hoy. GRE llega en v2 antes de julio 2026.

## Componentes

| Carpeta | Qué hace | Stack |
|---------|----------|-------|
| `apps/api` | API REST pública, auth, persistencia, cola de jobs | Go + chi + PostgreSQL + Redis |
| `apps/motor` | Genera UBL 2.1, firma XAdES-BES, habla SOAP con SUNAT, parsea CDR | PHP + Greenter |
| `apps/web` | Interfaz de usuario, PWA offline-first | React + TypeScript + Vite + Tailwind |
| `packages/codigos-sunat` | Catálogos SUNAT (tipos de documento, monedas, unidades, etc.) | JSON estático |
| `packages/ubl-schemas` | XSDs de UBL 2.1 que SUNAT exige | XML |

## Levantar en local (homologación / beta)

```bash
cp .env.example .env
# editar .env con los datos del tenant inicial
docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d
```

Más detalles en `docs/instalacion.md`.

## Antes de emitir tu primer comprobante

1. Obtené tu **Certificado Digital Tributario (CDT)** — `docs/obtener-cdt.md`.
2. Creá un **usuario secundario Clave SOL** con permiso de Emisión de CPE — `docs/crear-usuario-secundario.md`.
3. Probá primero contra el endpoint **beta (homologación)** de SUNAT antes de tocar producción.

## Roadmap corto

- [x] Decisión de stack del motor (ADR 0001).
- [x] Scaffolding del monorepo.
- [x] Docker Compose mínimo.
- [ ] Primer "hola mundo": emitir 1 factura contra SUNAT beta y guardar el CDR.
- [ ] Flujo completo factura + boleta + notas de crédito/débito.
- [ ] PDF bonito con QR.
- [ ] PWA offline-first con cola local.
- [ ] GRE (v2, antes de julio 2026).
