// Inicialización de features móviles cuando la app corre dentro de
// Capacitor (envoltorio nativo). En web puro estas llamadas son no-ops.

import { Capacitor } from "@capacitor/core";

export async function setupMobileAppearance(): Promise<void> {
  if (!Capacitor.isNativePlatform()) return;
  try {
    const { StatusBar, Style } = await import("@capacitor/status-bar");
    await StatusBar.setStyle({ style: Style.Dark });
    await StatusBar.setBackgroundColor({ color: "#0f172a" });
  } catch {
    // Plugin no disponible — no es crítico.
  }
  try {
    const { SplashScreen } = await import("@capacitor/splash-screen");
    await SplashScreen.hide({ fadeOutDuration: 200 });
  } catch { /* noop */ }
}

// Vibración corta para feedback de éxito.
export async function hapticOK(): Promise<void> {
  if (!Capacitor.isNativePlatform()) return;
  try {
    const { Haptics, ImpactStyle } = await import("@capacitor/haptics");
    await Haptics.impact({ style: ImpactStyle.Light });
  } catch { /* noop */ }
}

// Vibración doble para error/atención.
export async function hapticError(): Promise<void> {
  if (!Capacitor.isNativePlatform()) return;
  try {
    const { Haptics, NotificationType } = await import("@capacitor/haptics");
    await Haptics.notification({ type: NotificationType.Error });
  } catch { /* noop */ }
}

// Mantener pantalla encendida (útil en mostrador / emisión continua).
export async function keepAwake(on: boolean): Promise<void> {
  if (!Capacitor.isNativePlatform()) return;
  try {
    const { KeepAwake } = await import("@capacitor-community/keep-awake");
    if (on) await KeepAwake.keepAwake();
    else await KeepAwake.allowSleep();
  } catch { /* noop */ }
}
