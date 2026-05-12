# Obtener tu Certificado Digital Tributario (CDT)

El CDT es el certificado que usa el sistema para firmar digitalmente cada CPE que emitís. Sin él, SUNAT rechaza todo.

## Características

- **Gratis.** SUNAT lo emite sin costo desde Clave SOL.
- **Vigencia: 3 años.** Después tenés que renovarlo (mismo procedimiento).
- **Formato:** archivo `.p12` (PKCS#12) protegido por una passphrase que vos elegís.
- **Uso:** firma de CPE únicamente. No sirve para otros trámites (RUC, declaraciones, etc.) — para eso está la firma de Clave SOL.

## Procedimiento

1. Entrá a **SUNAT Operaciones en Línea (SOL)** con tu Clave SOL: <https://e-menu.sunat.gob.pe/cl-ti-itmenu/MenuInternet.htm>.
2. Andá al menú: **Empresas → Comprobantes de Pago → SEE - Del Contribuyente → Certificado Digital Tributario (CDT)**.
3. Hacé click en **"Solicitar Certificado Digital Tributario"**.
4. SUNAT te va a pedir que definas una **contraseña/passphrase** para proteger el `.p12`. **Anotala en un lugar seguro** — sin ella el certificado no sirve.
5. Descargá el archivo `.p12` (a veces se llama `<RUC>.p12` o similar).
6. Guardalo en un lugar seguro. **No lo subas a Dropbox / Drive / GitHub.** Si alguien lo tiene junto con la passphrase, puede firmar comprobantes en tu nombre.

## Por qué SUNAT te lo da gratis

Hace algunos años había que comprar certificados a entidades emisoras (Camerfirma, IDenTrust, etc.) por S/ 200-500 por año. SUNAT, harta de que esto fuera barrera de entrada, lanzó el CDT propio. Tomalo y usalo.

## Mitos comunes

- **"Tengo que renovarlo cada año."** Falso: 3 años.
- **"Si pierdo el `.p12` ya no puedo facturar."** Falso: lo volvés a generar desde Clave SOL. Eso sí, el nuevo `.p12` tiene una passphrase nueva que vos definís.
- **"Necesito un certificado de Camerfirma o similar."** Falso: el CDT de SUNAT es suficiente para emitir CPE.

## Subirlo a este sistema

Copiá el `.p12` a la carpeta `certs/` de la instalación. La aplicación te va a pedir la passphrase cada vez que reinicies el container (no se persiste — es a propósito, máxima soberanía).
