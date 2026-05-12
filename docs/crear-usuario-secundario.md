# Crear usuario secundario Clave SOL para emisión de CPE

SUNAT no quiere que uses tu Clave SOL principal (la que usás vos mismo para entrar al portal) para que un sistema automatizado emita comprobantes. En cambio, te exige crear un **usuario secundario** con permisos limitados solo para esa función.

Si no creás este usuario secundario, vas a recibir el error `0111 - No tiene el perfil para enviar comprobantes electrónicos` cuando intentes enviar tu primer CPE.

## Procedimiento

1. Entrá a Clave SOL con tu **usuario principal** (titular del RUC o representante legal).
2. Andá a: **Empresas → Mi RUC y Otros Registros → Administración de Usuarios Secundarios → Registro de Usuario Secundario**.
3. Hacé click en **"Crear Nuevo Usuario"**.
4. Llená:
   - **Tipo de usuario:** "Trabajador" o el rol que corresponda. Para uso del facturador, "Trabajador" funciona.
   - **Apellidos y nombres:** el nombre real del trabajador, o un nombre referencial si es para un sistema (ej. "FACTURADOR SISTEMA").
   - **Tipo de documento:** DNI o el que corresponda. Si es referencial, podés usar el del responsable.
   - **Usuario:** un nombre de usuario corto, ej. `FACTURADOR`. Quedará como `20XXXXXXXXX-FACTURADOR` (el RUC va adelante).
   - **Clave:** una contraseña fuerte. **Anotala** — la vas a necesitar en el `.env` o en la config del tenant.
5. En la sección de **perfiles**, asignale al menos:
   - **"Emisión electrónica de comprobantes desde los sistemas del contribuyente"** (también listado como "Facturador SUNAT SOL" en algunos menús).
6. Guardar.

## Verificación

Hacé logout y entrá con el usuario secundario para verificar que entra. No te va a dejar hacer todo, pero al menos deberías poder loguearte.

Después, configuralo en el sistema (al crear el tenant en la UI o vía API). El sistema lo usa para autenticarse contra el WS de SUNAT.

## Diferencias entre usuario principal y secundario

| | Principal | Secundario para CPE |
|---|---|---|
| Quién lo crea | SUNAT al registrar el RUC | Vos, desde Clave SOL |
| Para qué sirve | Trámites del RUC, declaraciones, todo | Solo lo que le asignes — en este caso: emitir CPE |
| Si lo comprometen | Catástrofe | Limitado al permiso de emisión |
| Va en el sistema | **NO** — nunca poner el principal | **Sí** — el sistema usa este |

**Regla de oro:** la Clave SOL principal nunca toca el sistema. Si la ves en un `.env`, en un archivo de config, o en cualquier lado de la aplicación, eso es un bug de seguridad.
