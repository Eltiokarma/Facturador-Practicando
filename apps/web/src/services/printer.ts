// Servicio de impresión BLE multi-plataforma.
//
//   - En app nativa (Capacitor Android): usa @capacitor-community/bluetooth-le
//     para descubrir, conectar e imprimir en impresoras térmicas BLE/Classic.
//   - En navegador (Chrome desktop o Chrome Android instalado como PWA):
//     usa Web Bluetooth API (BluetoothDevice → GATT).
//   - En navegadores sin soporte (Safari, Firefox): no soportado — la app
//     muestra mensaje claro y deja descargar el PDF como fallback.
//
// La impresora "favorita" se persiste en localStorage para que el cajero
// no tenga que volver a elegir cada vez.

import { Capacitor } from "@capacitor/core";

const LS_FAV = "facturador.printer.device";
const LS_AUTO = "facturador.printer.auto";
const LS_COLS = "facturador.printer.cols";

/** Configuración del cajero respecto a la impresora. */
export type PrinterSettings = {
  /** Si true, al recibir 'aceptado' de SUNAT se imprime automáticamente. */
  auto: boolean;
  /** Ancho del papel en caracteres. 42 = 80mm (común), 32 = 58mm. */
  cols: 32 | 42;
};

export function getPrinterSettings(): PrinterSettings {
  const auto = localStorage.getItem(LS_AUTO) === "1";
  const cols = localStorage.getItem(LS_COLS) === "32" ? 32 : 42;
  return { auto, cols };
}

export function savePrinterSettings(s: PrinterSettings): void {
  localStorage.setItem(LS_AUTO, s.auto ? "1" : "0");
  localStorage.setItem(LS_COLS, String(s.cols));
}

// UUID estándar de servicio de impresoras térmicas ESC/POS (la mayoría
// de fabricantes chinos usan este servicio "Serial Port Profile" sobre BLE).
const PRINTER_SERVICE_UUID = "000018f0-0000-1000-8000-00805f9b34fb";
const PRINTER_WRITE_CHAR   = "00002af1-0000-1000-8000-00805f9b34fb";

export type PrinterDevice = {
  id: string;       // address en Capacitor, id en Web Bluetooth
  name: string;
};

export type PrinterCapability =
  | "native"           // Capacitor BLE plugin disponible
  | "web-bluetooth"    // Web Bluetooth API en navegador
  | "unsupported";     // ni uno ni el otro

export function detectCapability(): PrinterCapability {
  if (Capacitor.isNativePlatform()) return "native";
  if (typeof navigator !== "undefined" && (navigator as any).bluetooth) {
    return "web-bluetooth";
  }
  return "unsupported";
}

export function getSavedPrinter(): PrinterDevice | null {
  try {
    const raw = localStorage.getItem(LS_FAV);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export function saveSavedPrinter(p: PrinterDevice | null) {
  if (!p) localStorage.removeItem(LS_FAV);
  else localStorage.setItem(LS_FAV, JSON.stringify(p));
}

// Pedir al usuario que seleccione una impresora. En cada plataforma usa el
// picker nativo correspondiente.
export async function pickPrinter(): Promise<PrinterDevice | null> {
  const cap = detectCapability();
  if (cap === "native") {
    const { BleClient } = await import("@capacitor-community/bluetooth-le");
    await BleClient.initialize({ androidNeverForLocation: true });
    const device = await BleClient.requestDevice({
      // Aceptamos cualquier dispositivo: las impresoras chinas no siempre
      // anuncian el servicio en su advertisement.
      services: [PRINTER_SERVICE_UUID],
      optionalServices: [PRINTER_SERVICE_UUID],
      namePrefix: undefined,
    });
    const p = { id: device.deviceId, name: device.name || "Impresora" };
    saveSavedPrinter(p);
    return p;
  }
  if (cap === "web-bluetooth") {
    const device = await navigator.bluetooth.requestDevice({
      filters: [{ services: [PRINTER_SERVICE_UUID] }],
      optionalServices: [PRINTER_SERVICE_UUID],
    });
    const p = { id: device.id, name: device.name || "Impresora" };
    saveSavedPrinter(p);
    return p;
  }
  throw new Error("Tu navegador no soporta Bluetooth. Probá con Chrome en Android o instalá la app.");
}

// Imprimir bytes ESC/POS. Reconecta con la impresora favorita si hace falta.
export async function printBytes(bytes: Uint8Array): Promise<void> {
  const cap = detectCapability();
  const fav = getSavedPrinter();
  if (!fav) {
    throw new Error("No hay impresora elegida. Andá a Configuración → Impresora.");
  }

  if (cap === "native") {
    const { BleClient } = await import("@capacitor-community/bluetooth-le");
    await BleClient.initialize({ androidNeverForLocation: true });
    try {
      await BleClient.connect(fav.id, undefined, { timeout: 10_000 });
      // Fragmentar en bloques (BLE MTU ~ 20-180 bytes según device).
      const chunkSize = 180;
      for (let i = 0; i < bytes.length; i += chunkSize) {
        const chunk = bytes.subarray(i, i + chunkSize);
        await BleClient.writeWithoutResponse(
          fav.id, PRINTER_SERVICE_UUID, PRINTER_WRITE_CHAR,
          new DataView(chunk.buffer, chunk.byteOffset, chunk.byteLength),
        );
      }
    } finally {
      try { await BleClient.disconnect(fav.id); } catch { /* ignore */ }
    }
    return;
  }

  if (cap === "web-bluetooth") {
    // Re-pedimos el device (Web Bluetooth no permite reconectar por id
    // entre sesiones; el usuario tiene que confirmar la primera vez).
    let device: BluetoothDevice;
    try {
      device = await navigator.bluetooth.requestDevice({
        filters: [{ services: [PRINTER_SERVICE_UUID] }],
        optionalServices: [PRINTER_SERVICE_UUID],
      });
    } catch (e) {
      throw new Error("No se pudo seleccionar la impresora: " + (e as Error).message);
    }
    const server = await device.gatt!.connect();
    try {
      const service = await server.getPrimaryService(PRINTER_SERVICE_UUID);
      const ch = await service.getCharacteristic(PRINTER_WRITE_CHAR);
      const chunkSize = 180;
      for (let i = 0; i < bytes.length; i += chunkSize) {
        const chunk = bytes.subarray(i, i + chunkSize);
        // Copia a un buffer concreto para que TS no se queje por ArrayBufferLike vs ArrayBuffer.
        const view = new Uint8Array(chunk.length);
        view.set(chunk);
        await ch.writeValueWithoutResponse(view);
      }
    } finally {
      server.disconnect();
    }
    return;
  }

  throw new Error("Bluetooth no soportado en este navegador.");
}
