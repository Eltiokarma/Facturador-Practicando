# ADR 0001 — Stack del motor de facturación

- **Fecha:** 2026-05-12
- **Estado:** Aceptada
- **Decisor:** Claude Code (delegado por el dueño del proyecto)

## Contexto

El "motor de facturación" es la pieza que:

1. Toma un JSON con los datos de un comprobante (factura, boleta, nota de crédito, nota de débito).
2. Lo serializa como XML UBL 2.1 según los esquemas SUNAT.
3. Lo firma digitalmente con XAdES-BES usando el `.p12` del contribuyente.
4. Lo empaqueta en un envelope SOAP 1.1 con WS-Security (UsernameToken + PasswordText).
5. Lo envía al endpoint de SUNAT correspondiente.
6. Parsea el CDR (Constancia de Recepción) que devuelve SUNAT, extrae el XML interno, valida.
7. Genera la representación impresa (PDF A4 + PDF ticket 80mm) con el QR del CPE.

Cada uno de esos pasos tiene aristas peruanas específicas: códigos de catálogo SUNAT, validaciones de RUC, redondeos de IGV, etc.

## Opciones evaluadas

### A — Go puro

Implementar todo en Go con `goxmldsig` o equivalente para XAdES.

- **Pro:** un solo binario, ecosistema unificado con el resto del backend.
- **Contra:** `goxmldsig` no resuelve XAdES-BES sobre UBL 2.1 sin trabajo importante. Hay que escribir templates UBL, normalización canónica, manejo de namespaces. Las librerías "sunat-go" que existen en GitHub son experimentales o incompletas.
- **Estimación:** 3-4x más tiempo a MVP versus B.

### B — Go API + microservicio PHP/Greenter (ELEGIDA)

El backend Go expone la API pública y la lógica de negocio (auth, validaciones, persistencia, cola). Llama internamente, vía HTTP en la red privada de Docker, a un microservicio PHP que corre Greenter para generar XML firmado, enviar a SUNAT y devolver el CDR.

- **Pro:** Greenter es la librería de facturación electrónica peruana más madura del ecosistema open source. Resuelve UBL 2.1 + XAdES + SOAP + PDF, está probada en producción por cientos de implementaciones. Reducimos riesgo técnico y aceleramos el MVP en semanas.
- **Pro:** mantiene Go como la cara visible del backend (alineado con el stack del POS del usuario).
- **Pro:** el motor PHP queda enjaulado en su propio container, sin acceso a red excepto al egress hacia los endpoints SUNAT. La superficie de ataque es minúscula.
- **Contra:** dos lenguajes en el stack. Un container PHP-FPM (o PHP-CLI con un mini server) adicional.
- **Contra:** dependencia transitiva del mantenimiento de Greenter. Mitigación: pineamos versión exacta, mantenemos un fork si fuera necesario, los XMLs generados son auditables.

### C — 100% PHP/Laravel + Greenter

- **Pro:** lo más rápido al MVP, una sola tecnología.
- **Contra:** rompe el stack Go que ya maneja el usuario. Onboarding de devs futuros se vuelve más caro. El POS que ya tiene es Go: queremos coherencia.

### D — Node/TypeScript con `node-sunat` o equivalente

Las librerías Node para SUNAT (`node-sunat`, `peru-billing`, etc.) no tienen ni el mantenimiento ni la cobertura de Greenter. Descartada.

## Decisión

**Opción B.** Go API + microservicio PHP/Greenter.

## Consecuencias

### Positivas

- MVP funcional en semanas, no meses.
- Riesgo técnico de firma digital y serialización UBL transferido a una librería madura.
- Aislamiento de seguridad: si Greenter o el motor PHP tienen un bug, está confinado a un container sin acceso a la DB ni al filesystem del host (salvo el volumen de certificados, montado read-only).
- El backend Go mantiene la lógica de negocio, la API pública y el control de identidad.

### Negativas

- Dos runtimes (Go + PHP).
- El contract entre Go y motor PHP debe ser estricto y versionado. Lo definimos en `apps/motor/CONTRACT.md`.
- Operadores del sistema deben monitorear dos servicios.

### Mitigaciones

- El motor PHP no tiene base de datos. Es stateless. Si se cae, se reinicia, sin pérdida.
- Los certificados se montan en el container del motor como volumen read-only, descifrados en memoria al recibir la passphrase desde el backend Go (que la tiene en memoria, nunca en disco ni en DB).
- Healthcheck del motor desde el backend Go: si falla, los jobs de la cola reintentan con backoff.

## Plan de salida

Si en el futuro Greenter deja de mantenerse o queremos consolidar en Go, la API entre `apps/api` y `apps/motor` es estable y minimal (un POST con el JSON del comprobante, una respuesta con XML firmado + CDR). Reemplazar el motor PHP por un motor Go (o cualquier otro lenguaje) implica reescribir solo `apps/motor`, sin tocar el resto.
