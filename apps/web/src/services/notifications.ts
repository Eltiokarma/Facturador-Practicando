// Notificaciones para avisar al cajero cuando SUNAT termina de responder
// un comprobante (que normalmente está minimizado o en otra pestaña).
//
//   - En app nativa Android: usa @capacitor/local-notifications. Funciona
//     aunque la app esté en background, suena, vibra, queda en la barra.
//   - En navegador: usa la Notification API estándar. Suficiente para
//     pestañas en segundo plano del mostrador.
//
// Usamos notificaciones LOCALES, no push. La diferencia: las locales las
// dispara la propia app desde adentro (cuando el polling detecta el
// cambio de estado). Las push requieren un servidor FCM y son overkill
// para este caso.

import { Capacitor } from "@capacitor/core";

type Permission = "granted" | "denied" | "default";

/** Pide permiso al usuario (idempotente). Llamar al iniciar la app. */
export async function requestPermission(): Promise<Permission> {
  if (Capacitor.isNativePlatform()) {
    try {
      const { LocalNotifications } = await import("@capacitor/local-notifications");
      const perm = await LocalNotifications.requestPermissions();
      return perm.display === "granted" ? "granted" : "denied";
    } catch {
      return "denied";
    }
  }
  if (typeof window !== "undefined" && "Notification" in window) {
    if (Notification.permission === "granted") return "granted";
    if (Notification.permission === "denied") return "denied";
    const r = await Notification.requestPermission();
    return r as Permission;
  }
  return "denied";
}

/** Dispara una notificación. Silencioso si el usuario no dio permiso. */
export async function notify(title: string, body: string): Promise<void> {
  if (Capacitor.isNativePlatform()) {
    try {
      const { LocalNotifications } = await import("@capacitor/local-notifications");
      await LocalNotifications.schedule({
        notifications: [{
          id: Date.now() % 2_147_483_647,
          title,
          body,
          smallIcon: "ic_stat_icon_config_sample",
          schedule: { at: new Date(Date.now() + 100) },
        }],
      });
    } catch { /* noop */ }
    return;
  }
  if (typeof window !== "undefined" && "Notification" in window
      && Notification.permission === "granted") {
    try {
      new Notification(title, { body, tag: "facturador" });
    } catch { /* noop */ }
  }
}
