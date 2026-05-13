import type { CapacitorConfig } from "@capacitor/cli";

const config: CapacitorConfig = {
  appId: "pe.facturador.app",
  appName: "Facturador",
  webDir: "dist",
  // Si server.url está definido, la app móvil se conecta a un servidor
  // remoto en vez de servir el bundle local. Útil para desarrollo
  // (HMR contra http://192.168.x.x:5173). En release no se setea.
  server: {
    androidScheme: "https",
  },
  android: {
    allowMixedContent: false,
    backgroundColor: "#0f172a",
  },
  plugins: {
    SplashScreen: {
      launchShowDuration: 800,
      backgroundColor: "#0f172a",
      androidSplashResourceName: "splash",
      androidScaleType: "CENTER_CROP",
      showSpinner: false,
      splashFullScreen: true,
      splashImmersive: true,
    },
    BluetoothLe: {
      displayStrings: {
        scanning: "Buscando impresoras…",
        cancel: "Cancelar",
        availableDevices: "Impresoras disponibles",
        noDeviceFound: "No se encontraron impresoras",
      },
    },
  },
};

export default config;
