# Explorar el sistema en modo DEMO

Para ver cómo funciona el facturador sin necesidad de conseguir certificado SUNAT ni usuario secundario Clave SOL. Tarda unos 5 minutos.

## 1. Tres secretos en `.env`

```bash
cd facturador
cp .env.example .env
$EDITOR .env
```

Solo cambiá estas tres variables. Lo demás dejá los defaults.

```env
POSTGRES_PASSWORD=algo-largo-y-aleatorio
API_JWT_SECRET=<copiar la salida de: openssl rand -hex 32>
MASTER_KEY=<copiar la salida de: openssl rand -hex 32>
```

Para generarlas en una sola línea:

```bash
echo "POSTGRES_PASSWORD=$(openssl rand -hex 16)"
echo "API_JWT_SECRET=$(openssl rand -hex 32)"
echo "MASTER_KEY=$(openssl rand -hex 32)"
```

**No** llenes nada de `TENANT_*` ni `CERT_*` — todo eso se configura después desde la UI.

## 2. Levantar el stack

```bash
docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d --build
```

La primera vez tarda unos minutos en bajar imágenes y compilar. Verificá que esté arriba:

```bash
curl http://localhost:8080/health
# {"status":"ok","sunat_mode":"beta","time":"..."}
```

## 3. Crear el primer usuario

```bash
docker compose exec api /app/seed-user \
    -email=tu@email -password=demodemo -nombre="Tu nombre" -rol=dueno
```

Como no hay tenants en DB y no definiste `TENANT_*`, el script crea automáticamente una empresa **EMPRESA DEMO** con RUC `20000000001`. Te asigna como dueño.

## 4. Entrar a la UI

Abrí `http://localhost:5173`, logueá con `tu@email` / `demodemo`.

Vas a ver:

- **Banner amarillo arriba**: "Esta empresa está en modo DEMO. Los comprobantes NO se envían a SUNAT."
- **Badge DEMO en el sidebar** al lado del RUC.

## 5. Explorar el flujo completo

Cosas que podés hacer sin tocar SUNAT:

### Emitir factura demo

1. Sidebar → **Emitir comprobante**.
2. Cargá un cliente (RUC + razón social).
3. Agregá ítems (descripción + cantidad + precio).
4. Click **Emitir**.
5. La UI te lleva al detalle. En 1-2 segundos cambia de "Enviando…" a **Aceptado** (simulado).
6. Click **Ver PDF**: vas a ver el PDF con marca de agua **DEMO** atravesando la página.

### Catálogos

- **Clientes**: cargá un par de clientes, después aparecen como autocomplete cuando emitís.
- **Productos**: igual con productos del menú/inventario.

### Notas de crédito y débito

1. Andá a una factura/boleta aceptada (en demo todas son aceptadas).
2. En la pantalla de detalle, abajo, vas a ver **Nota de crédito** y **Nota de débito**.
3. Click → form con el motivo SUNAT (catálogo 09 o 10) e ítems prellenados desde el comprobante original.
4. Editá lo que querés ajustar y emití.

### Resumen diario

1. Emití varias **boletas** (tipo 03) con la misma fecha.
2. Sidebar → **Resumen diario**.
3. Elegí la fecha, vas a ver todas las boletas listadas.
4. Click **Enviar resumen**. En demo se acepta sin contactar SUNAT.

### Multi-empresa

1. En el selector del sidebar (arriba), click "+ Nueva empresa".
2. Cargá RUC + razón social.
3. Quedás como dueño de la nueva empresa, también en modo DEMO.
4. Usá el selector para saltar entre las dos.

## 6. Pasar de DEMO a real (cuando estés listo)

1. Conseguí tu **CDT** (`.p12`) de Clave SOL — ver `docs/obtener-cdt.md`.
2. Conseguí un **usuario secundario** Clave SOL con permiso de emisión — ver `docs/crear-usuario-secundario.md`.
3. En la UI, sidebar → **Configuración**.
4. Actualizá razón social y demás datos para que coincidan con SUNAT.
5. En "Certificado Digital Tributario": subí el `.p12` y la passphrase.
6. En "Credenciales SUNAT": usuario y clave del usuario secundario.
7. Cuando todo esté cargado, destildá **Modo DEMO** y guardá.
8. Probá emitir una factura: ahora sí va a SUNAT beta.
9. Cuando funcione en beta, en Configuración cambiá modo de `beta` a `prod`.

## Limpiar todo y empezar de cero

```bash
docker compose down -v   # borra volúmenes (DB + Redis)
rm -rf certs/ data/      # borra certs y artefactos
```

## Qué hace exactamente el modo DEMO por dentro

- El worker no llama al motor PHP/Greenter.
- Simula la respuesta de SUNAT con `estado: aceptado`, `codigo: 0`, mensaje claro de DEMO.
- El hash del CPE empieza con `DEMO-` para que sea inequívoco.
- El PDF se genera igual, pero con marca de agua **DEMO** rotada en diagonal.
- No se genera XML firmado ni CDR (no hay cert ni respuesta real).
- En la lista de comprobantes, todos los DEMO aparecen como aceptados con el mensaje correspondiente.

Si emitís en DEMO y después salís a producción, esos comprobantes históricos quedan marcados como DEMO en su mensaje SUNAT, así que no se confunden con los reales.
