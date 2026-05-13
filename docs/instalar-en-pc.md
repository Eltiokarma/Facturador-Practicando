# Instalar el Facturador en tu PC — guía para no-programadores

Al final de esto, vas a tener un **ícono "Facturador" en tu escritorio** que al hacer doble click abre la app, como cualquier otra. La primera vez tarda 15 minutos. Las siguientes, dos clicks.

## Lo que vas a necesitar

| Programa | Para qué | Cuánto pesa |
|---|---|---|
| **Docker Desktop** | Corre el motor del Facturador en segundo plano | ~500 MB |
| **Git** | Bajar y actualizar el código del Facturador | ~80 MB |
| **Google Chrome** | Para usar la app (Edge también sirve) | ya lo tenés probablemente |

Si tu compu tiene menos de 4 GB de RAM o un disco lleno, no va a andar bien — el Facturador necesita ~2 GB libres.

---

## Paso 1 — Instalar Docker Desktop

### Windows o Mac

1. Andá a **docker.com/products/docker-desktop**.
2. Click **Download for Windows** o **Download for Mac**.
3. Abrí el instalador, siguiente, siguiente.
4. En Windows te puede pedir habilitar **WSL2** durante la instalación. Aceptá y dejá que reinicie.
5. Después de instalar, abrí **Docker Desktop**. La primera vez tarda un par de minutos en arrancar.
6. Cuando ves el ícono de la **ballena 🐳 fija** en la barra (Windows: bandeja del sistema abajo a la derecha; Mac: arriba a la derecha), Docker está listo.

### Linux

Instalá Docker Engine + Docker Compose desde el gestor de paquetes de tu distro o seguí la guía oficial en docs.docker.com/engine/install. Agregate al grupo `docker`:

```bash
sudo usermod -aG docker $USER
```

Y volvé a iniciar sesión.

---

## Paso 2 — Instalar Git

### Windows

1. Andá a **git-scm.com**.
2. Click **Download for Windows**.
3. Abrí el instalador y dejá todo por defecto. Siguiente hasta que termine.

### Mac

Abrí la **Terminal** (busca "Terminal" en Spotlight) y pegá:

```bash
xcode-select --install
```

Te pregunta si querés instalar las herramientas de línea de comandos. Aceptá. Tarda 2-5 minutos.

### Linux

```bash
sudo apt install git    # Ubuntu/Debian
sudo dnf install git    # Fedora
```

---

## Paso 3 — Bajar el Facturador a tu compu

Abrí la terminal:
- **Windows**: tocá la tecla **Windows**, escribí `powershell`, abrí.
- **Mac**: Spotlight (Cmd+Espacio), escribí `terminal`, abrí.
- **Linux**: ya sabés.

Pegá esto (línea por línea o todo junto, da igual):

```bash
cd ~/Documentos
git clone https://github.com/Eltiokarma/Facturador-Practicando.git facturador
cd facturador
git checkout claude/setup-peru-facturador-JsiRS
```

> ¿Cambió tu carpeta `Documentos`? En Mac/Linux se llama `~/Documents` o `~/Documentos` según el idioma del sistema. En Windows está en `C:\Users\TuUsuario\Documents`. Si no estás seguro, dejá `cd ~` (carpeta de usuario).

Eso te deja una carpeta `facturador/` con todo adentro.

---

## Paso 4 — Arrancar el Facturador la primera vez

### En Windows

1. Abrí el **Explorador de archivos** y andá a `Documentos\facturador\scripts\`.
2. Doble click en **`arrancar-windows.bat`**.
3. Se abre una ventana negra con texto verde. **No la cierres** — está trabajando.
4. La primera vez tarda 3-5 minutos. Vas a ver mensajes tipo:
   - `Levantando servicios (la primera vez tarda unos minutos)...`
   - Después aparecen líneas con `[+] Pulling...` y `[+] Building...`
5. Cuando ves **`✓ Facturador listo en http://localhost:5173`**, listo.
6. Se abre Chrome automáticamente con la app.
7. La ventana negra ya se puede cerrar tranquilo.

> Si te dice **"Docker no está corriendo"**: andá a la bandeja del sistema (esquina inferior derecha), abrí Docker Desktop, esperá a que diga "Engine running", y volvé a hacer doble click en el .bat.

### En Mac

1. Abrí **Finder** y andá a `Documentos/facturador/scripts/`.
2. Doble click en **`arrancar-mac.command`**.
3. La primera vez Mac te dice **"No se puede abrir porque no se puede verificar el desarrollador"**. Es normal — el archivo es tuyo, no firmado. Solución:
   - Click derecho sobre `arrancar-mac.command` → **Abrir** → **Abrir** (en el diálogo de advertencia).
   - Solo hay que hacerlo una vez. Las próximas, doble click normal.
4. Se abre una ventana de Terminal. Tarda 3-5 minutos compilando.
5. Cuando termina, abre Chrome con la app.
6. La ventana de Terminal se puede cerrar.

### En Linux

1. Abrí tu gestor de archivos en `~/Documentos/facturador/scripts/`.
2. Click derecho sobre `arrancar-linux.sh` → **Propiedades** → **Permisos** → **Permitir ejecutar el archivo como un programa**.
3. Doble click. Algunas distros te preguntan "¿Ejecutar en terminal?" — sí.
4. Espera, mismo flow.

---

## Paso 5 — Loguearte por primera vez

Cuando Chrome se abre, te muestra la pantalla de login.

```
Usuario:    demo@local
Contraseña: demodemo
```

Esos son los datos generados automáticamente por el script. Te recomiendo cambiarlos después por algo tuyo (creando otro usuario desde la terminal, pero eso ya lo vemos cuando lo necesites).

Una vez adentro vas a ver el banner amarillo **"Modo DEMO"** — significa que las emisiones se simulan, no llegan a SUNAT. Perfecto para que pruebes todo sin romper nada.

---

## Paso 6 — Instalar como app de escritorio (el ícono que querés)

Esta es la parte mágica. Chrome puede convertir cualquier web en una app standalone.

### En Chrome desktop (Windows / Mac / Linux)

1. Con la app del Facturador abierta en Chrome, **mirá la barra de URL** (donde está `localhost:5173`).
2. A la derecha de la URL, en versiones recientes vas a ver un **ícono con una flechita y un monitor**, o un ícono de cuadradito con flecha. Tooltip dice: **"Instalar Facturador"** o **"Instalar aplicación"**.
3. Si no aparece ahí: menú **⋮** (tres puntitos arriba a la derecha) → **Transmitir, guardar y compartir** → **Instalar Facturador…**
4. Te muestra un diálogo confirmando. Click **Instalar**.
5. Listo: el Facturador se abre en una **ventana propia**, sin barra de URL, como una app normal.
6. **Windows**: aparece un ícono en el menú Inicio y en el escritorio (te pregunta dónde anclar).
7. **Mac**: aparece en Launchpad y podés arrastrarlo al Dock.
8. **Linux**: aparece en el menú de aplicaciones.

### En Edge

Mismo flujo: hay un ícono **"+"** o **"Instalar app"** en la barra de URL.

### En Firefox o Safari

Estos navegadores **no soportan instalar PWAs como apps de escritorio** (limitación de Mozilla y Apple). Tenés que usar Chrome o Edge.

---

## Flujo de uso diario (después del primer setup)

1. **Mañana**: doble click en `arrancar-windows.bat` (o el `.command` en Mac). Ventana negra → se cierra solita en 30 segundos cuando el sistema ya estaba compilado.
2. Doble click en el **ícono Facturador** del escritorio.
3. Trabajás todo el día.
4. **Noche**: cerrás la ventana del Facturador. Si querés liberar memoria de la compu, doble click en `apagar-windows.bat`.

Los datos quedan guardados aunque apagues. Cuando volvés a arrancar, todo está donde lo dejaste.

---

## Apagar/encender la compu sin perder datos

**Sí**, podés reiniciar tu PC tranquilo. Docker mantiene el estado en disco.

Si querés que el Facturador arranque automáticamente con la compu:

### Windows

1. Tocá **Windows + R**, escribí `shell:startup`, Enter. Te abre la carpeta de inicio.
2. Click derecho dentro → **Nuevo → Acceso directo** → pegá la ruta de `arrancar-windows.bat`. Siguiente, Finalizar.

### Mac

**Preferencias del Sistema → Usuarios y Grupos → Inicio de sesión → +** y agregás `arrancar-mac.command`.

---

## Si algo se rompe

| Síntoma | Qué hacer |
|---|---|
| Doble click no hace nada | Confirmá que Docker Desktop esté abierto (ícono ballena fijo). Volvé a hacer doble click. |
| "Puerto 5173 ya en uso" | Otra app usa ese puerto. Apagá esa app o reiniciá la compu. |
| El ícono de instalar no aparece en Chrome | Probá en una **ventana de incógnito**: Ctrl+Shift+N → `http://localhost:5173` → ahora sí aparece. Si solo aparece ahí, instalá desde la incógnita. |
| "no se pudo conectar" en la app instalada | Pasá el cursor sobre la ventana de la app, click derecho → **Recargar**. Si persiste, abrí el .bat de arrancar de nuevo. |
| La compu va lenta cuando está el Facturador | Es Docker consumiendo RAM. En **Docker Desktop → Settings → Resources** podés bajarle la RAM asignada (default 8 GB; con 2 GB sobra). |
| Otro error | Hacé screenshot de la ventana negra/terminal y mandámela. |

---

## Actualizar el Facturador a la versión nueva

Cuando saquemos cambios:

```bash
cd ~/Documentos/facturador
git pull
```

Después doble click en `arrancar-*` de vuelta. Va a recompilar solo los servicios que cambiaron. Si la base de datos cambió, las migraciones se aplican automáticamente.
