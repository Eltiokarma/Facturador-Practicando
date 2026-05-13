# App móvil (Android)

El frontend del facturador funciona como:

1. **Web normal** en cualquier navegador moderno (Chrome, Edge, Firefox, Safari).
2. **PWA instalable** ("Agregar a pantalla de inicio") — se ve como app, pero corre en el motor del navegador.
3. **App Android nativa** (APK / AAB), empaquetada con [Capacitor](https://capacitorjs.com).

La opción 3 es la recomendada para uso real en mostrador, especialmente si vas a **imprimir tickets en una impresora térmica Bluetooth**.

## ¿Por qué la app nativa?

- **Bluetooth Classic + BLE** confiable en Android (incluyendo impresoras Xprinter, Bixolon, EPSON TM, etc.). En navegador funciona en Chrome pero no en Safari/Firefox.
- **Pantalla siempre encendida** durante emisión (cajero no se pelea con el bloqueo).
- **Vibración táctil** al confirmar emisión.
- **Splash screen** con la marca al abrir.
- Se ve como cualquier otra app del cajón del Android — confianza para el cajero y el dueño.

## Cómo buildear la APK

### Una sola vez

Instalá en tu máquina:

- [Node.js 20+](https://nodejs.org)
- [Android Studio](https://developer.android.com/studio) (te baja el Android SDK y el JDK)
- Aceptá las licencias del SDK:
  ```bash
  sdkmanager --licenses
  ```

### Cada vez que querés generar APK

```bash
cd apps/web

# Instalar deps (la primera vez o tras pull)
npm install

# Compilar el frontend
npm run build

# Generar el proyecto Android la primera vez (queda en apps/web/android/)
npx cap add android

# Sincronizar el bundle web con el proyecto Android
npx cap sync android

# Abrir Android Studio para buildear / firmar
npx cap open android
```

En Android Studio: **Build → Build Bundle(s) / APK(s) → Build APK(s)**. La APK queda en `apps/web/android/app/build/outputs/apk/debug/app-debug.apk`.

Para firmar para Play Store: **Build → Generate Signed Bundle / APK** y seguís el wizard (te genera el `.aab` para subir a Google Play Console).

### Configurar la URL del backend

Por defecto la app móvil sirve el bundle web local. Si querés que el cajero apunte a tu API en producción:

Editá `apps/web/capacitor.config.ts` antes de hacer `cap sync`:

```ts
server: {
  url: "https://facturador.tuempresa.pe",
  cleartext: false,
}
```

O dejá la URL en `apps/web/.env.production` con `VITE_API_BASE_URL=https://...` y rebuildeás.

## Imprimir tickets

1. Después de loguearte en la app, andá a **Configuración → Impresora térmica**.
2. Tocá **Emparejar impresora**. Android te muestra los Bluetooth disponibles. Elegí tu térmica (en general el nombre incluye "Printer" o el modelo).
3. Probá con **Imprimir prueba** para verificar.
4. A partir de ahí, cada comprobante aceptado tiene un botón **🖨 Imprimir ticket** que envía el ESC/POS automáticamente.

### Impresoras testeadas

Las impresoras térmicas más comunes en Perú compatibles con el modo ESC/POS estándar:

- Xprinter XP-58, XP-80 (Bluetooth/USB).
- Bixolon SRP-330II, SPP-R200/R210/R310/R410.
- EPSON TM-T20, TM-m30 (con módulo BT).
- Genéricas chinas tipo "MPT-II" / "PT-210" (las del mercado).

Si tu impresora no aparece en el picker, asegurate de:

1. Que esté **emparejada en los ajustes Bluetooth de Android** previamente.
2. Tener el **plástico de la cinta** retirado (algunas vienen con un seguro).
3. Que esté **encendida** (botón largo hasta el LED fijo).

## Distribución

Para uso interno (1 restaurante): la APK firmada se instala desde el archivo, no necesitás Play Store.

Para distribución amplia: subí el `.aab` a Google Play Console como app interna o abierta. Requiere cuenta de desarrollador Google (USD 25 una sola vez).

## Lo que dejé fuera de esta versión

- **iOS** (`npx cap add ios`): la base ya soporta, pero requiere macOS + Xcode para buildear. Se agrega cuando aparezca el primer cliente que lo pida.
- **Notificaciones push**: cuando SUNAT acepta/rechaza, notificar al cajero. Plugin disponible, todavía no cableado.
- **Cámara para escanear QR**: para validar comprobantes que recibe el dueño. Plugin disponible.
- **Offline real con cola en IndexedDB**: hoy si se cae internet la emisión falla. La cola server-side queda intacta, pero el cajero ve el error. Falta replicar la cola en el cliente.
