import { FormEvent, useEffect, useState } from "react";
import { toast } from "sonner";
import { api } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import {
  PrinterDevice,
  PrinterSettings,
  detectCapability,
  getPrinterSettings,
  getSavedPrinter,
  pickPrinter,
  printBytes,
  savePrinterSettings,
  saveSavedPrinter,
} from "../services/printer";
import { EscPos } from "../services/escpos";

type TenantConfig = {
  id: string;
  ruc: string;
  razon_social: string;
  nombre_comercial?: string;
  direccion_fiscal?: string;
  ubigeo?: string;
  sunat_mode: "beta" | "prod";
  usuario_sol?: string;
  cert_path?: string;
  has_cert?: boolean;
  demo_mode: boolean;
  gre_client_id?: string;
  has_gre_credenciales?: boolean;
};

export function Configuracion() {
  const { user } = useAuth();
  const [t, setT] = useState<TenantConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [savingPerfil, setSavingPerfil] = useState(false);

  // Credenciales SOL
  const [usuarioSOL, setUsuarioSOL] = useState("");
  const [claveSOL, setClaveSOL] = useState("");
  const [savingCreds, setSavingCreds] = useState(false);

  const esDueno = user?.rol === "dueno";

  useEffect(() => {
    api<TenantConfig>("/api/v1/tenant")
      .then((d) => {
        setT(d);
        setUsuarioSOL(d.usuario_sol || "");
      })
      .catch(() => toast.error("No se pudo cargar la configuración"))
      .finally(() => setLoading(false));
  }, []);

  if (loading || !t) return <div className="text-slate-500 text-sm">Cargando…</div>;

  async function guardarPerfil(e: FormEvent) {
    e.preventDefault();
    if (!t) return;
    setSavingPerfil(true);
    try {
      await api("/api/v1/tenant", {
        method: "PUT",
        body: {
          razon_social: t.razon_social,
          nombre_comercial: t.nombre_comercial,
          direccion_fiscal: t.direccion_fiscal,
          ubigeo: t.ubigeo,
          sunat_mode: t.sunat_mode,
          demo_mode: t.demo_mode,
        },
      });
      toast.success("Datos guardados.");
      window.location.reload();
    } catch (e: any) {
      toast.error(e?.body?.detalle || "No se pudo guardar");
    } finally {
      setSavingPerfil(false);
    }
  }

  async function guardarCreds(e: FormEvent) {
    e.preventDefault();
    if (!usuarioSOL || !claveSOL) {
      toast.error("Usuario y clave SOL son obligatorios");
      return;
    }
    setSavingCreds(true);
    try {
      await api("/api/v1/tenant/credenciales", {
        method: "PUT",
        body: { usuario_sol: usuarioSOL, clave_sol: claveSOL },
      });
      toast.success("Credenciales SOL actualizadas. Reiniciá el servicio para que surtan efecto.");
      setClaveSOL("");
    } catch (e: any) {
      toast.error(e?.body?.detalle || "No se pudo guardar");
    } finally {
      setSavingCreds(false);
    }
  }

  return (
    <div className="space-y-8">
      <header>
        <h1 className="text-2xl font-semibold text-slate-900">Configuración</h1>
        <p className="text-sm text-slate-500 mt-1">
          Datos de tu empresa, credenciales SUNAT y modo de emisión.
        </p>
      </header>

      {!esDueno && (
        <div className="rounded-lg p-4 text-sm bg-amber-50 border border-amber-200 text-amber-800">
          Solo los usuarios con rol <strong>dueño</strong> pueden modificar la configuración.
        </div>
      )}

      {t.demo_mode && (
        <div className="rounded-lg p-4 text-sm bg-amber-50 border border-amber-200 text-amber-900">
          <strong>Esta empresa está en modo DEMO.</strong> Los comprobantes se
          simulan localmente y no llegan a SUNAT. Cuando tengas tu certificado
          <code>.p12</code> y credenciales SOL configurados, podés desactivarlo abajo.
        </div>
      )}

      <section className="card p-6">
        <h2 className="text-lg font-medium text-slate-900 mb-1">Identidad de la empresa</h2>
        <p className="text-xs text-slate-500 mb-4">
          Debe coincidir exactamente con lo que SUNAT tiene registrado.
        </p>
        <form onSubmit={guardarPerfil} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">RUC</label>
              <input className="input bg-slate-50" value={t.ruc} disabled readOnly />
              <p className="text-xs text-slate-400 mt-1">No editable — está fijado al alta.</p>
            </div>
            <div>
              <label className="label">Modo SUNAT</label>
              <select
                className="input"
                value={t.sunat_mode}
                onChange={(e) => setT({ ...t, sunat_mode: e.target.value as any })}
                disabled={!esDueno}
              >
                <option value="beta">Beta (homologación)</option>
                <option value="prod">Producción</option>
              </select>
              {t.sunat_mode === "prod" && (
                <p className="text-xs text-amber-700 mt-1">
                  ⚠ En modo producción, los comprobantes tienen valor legal real.
                </p>
              )}
            </div>
          </div>
          <div>
            <label className="label">Razón social</label>
            <input
              className="input"
              value={t.razon_social}
              onChange={(e) => setT({ ...t, razon_social: e.target.value })}
              disabled={!esDueno}
              required
            />
          </div>
          <div>
            <label className="label">Nombre comercial</label>
            <input
              className="input"
              value={t.nombre_comercial || ""}
              onChange={(e) => setT({ ...t, nombre_comercial: e.target.value })}
              disabled={!esDueno}
            />
          </div>
          <div>
            <label className="label">Dirección fiscal</label>
            <input
              className="input"
              value={t.direccion_fiscal || ""}
              onChange={(e) => setT({ ...t, direccion_fiscal: e.target.value })}
              disabled={!esDueno}
            />
          </div>
          <div>
            <label className="label">Ubigeo (6 dígitos)</label>
            <input
              className="input max-w-xs"
              value={t.ubigeo || ""}
              maxLength={6}
              onChange={(e) => setT({ ...t, ubigeo: e.target.value.replace(/\D/g, "") })}
              disabled={!esDueno}
            />
          </div>
          <div className="pt-3 border-t border-slate-100">
            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="checkbox"
                className="mt-0.5"
                checked={t.demo_mode}
                onChange={(e) => setT({ ...t, demo_mode: e.target.checked })}
                disabled={!esDueno}
              />
              <span>
                <span className="text-sm font-medium text-slate-900 block">Modo DEMO</span>
                <span className="text-xs text-slate-500">
                  Las emisiones se simulan localmente sin enviarse a SUNAT. Ideal
                  para explorar el sistema o probar el flujo sin certificado real.
                  Para desactivarlo, primero subí el .p12 abajo.
                </span>
              </span>
            </label>
          </div>
          {esDueno && (
            <div className="flex justify-end pt-2">
              <button type="submit" className="btn-primary" disabled={savingPerfil}>
                {savingPerfil ? "Guardando…" : "Guardar datos"}
              </button>
            </div>
          )}
        </form>
      </section>

      <section className="card p-6">
        <h2 className="text-lg font-medium text-slate-900 mb-1">Credenciales SUNAT (Clave SOL)</h2>
        <p className="text-xs text-slate-500 mb-4">
          Usuario secundario Clave SOL con permiso <em>Emisión electrónica de
          comprobantes desde los sistemas del contribuyente</em>. NUNCA usar la
          Clave SOL principal. La clave se guarda cifrada con AES-256-GCM.
        </p>
        <form onSubmit={guardarCreds} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">Usuario SOL</label>
              <input
                className="input"
                value={usuarioSOL}
                onChange={(e) => setUsuarioSOL(e.target.value)}
                placeholder="FACTURADOR"
                disabled={!esDueno}
              />
            </div>
            <div>
              <label className="label">Clave SOL</label>
              <input
                type="password"
                className="input"
                value={claveSOL}
                onChange={(e) => setClaveSOL(e.target.value)}
                placeholder="Dejá en blanco para no cambiar"
                disabled={!esDueno}
              />
            </div>
          </div>
          {esDueno && (
            <div className="flex justify-end pt-2">
              <button type="submit" className="btn-primary" disabled={savingCreds}>
                {savingCreds ? "Guardando…" : "Actualizar credenciales"}
              </button>
            </div>
          )}
        </form>
      </section>

      <CertSection tenant={t} esDueno={esDueno} onUploaded={() => window.location.reload()} />

      <GRESection tenant={t} esDueno={esDueno} />

      <PrinterSection />
    </div>
  );
}

function GRESection({ tenant: t, esDueno }: { tenant: TenantConfig; esDueno: boolean }) {
  const [clientId, setClientId] = useState(t.gre_client_id || "");
  const [clientSecret, setClientSecret] = useState("");
  const [saving, setSaving] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!clientId || !clientSecret) {
      toast.error("Client ID y Client Secret son obligatorios");
      return;
    }
    setSaving(true);
    try {
      await api("/api/v1/tenant/gre-credenciales", {
        method: "PUT",
        body: { client_id: clientId, client_secret: clientSecret },
      });
      toast.success("Credenciales API GRE guardadas (cifradas).");
      setClientSecret("");
    } catch (e: any) {
      toast.error(e?.body?.detalle || "No se pudo guardar");
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="card p-6">
      <h2 className="text-lg font-medium text-slate-900 mb-1">
        Credenciales API GRE (OAuth2)
      </h2>
      <p className="text-xs text-slate-500 mb-4">
        Solo necesarias si vas a emitir Guías de Remisión Electrónica (tipo 09).
        SUNAT exige Client ID + Client Secret distintos de Clave SOL. Los
        obtenés desde Clave SOL → Empresas → API SUNAT. Se guardan cifrados
        con AES-256-GCM.
      </p>
      <div className={`mb-4 rounded-lg p-3 text-sm border ${
        t.has_gre_credenciales
          ? "bg-emerald-50 border-emerald-200 text-emerald-800"
          : "bg-slate-50 border-slate-200 text-slate-700"
      }`}>
        {t.has_gre_credenciales
          ? "✓ Credenciales API GRE configuradas. Podés emitir guías."
          : "ℹ Sin credenciales API GRE configuradas. Solo necesarias si vas a usar GRE en modo real."}
      </div>
      {esDueno && (
        <form onSubmit={onSubmit} className="space-y-4">
          <div>
            <label className="label">Client ID</label>
            <input
              className="input"
              value={clientId}
              onChange={(e) => setClientId(e.target.value)}
              placeholder="UUID que te dio SUNAT"
            />
          </div>
          <div>
            <label className="label">Client Secret</label>
            <input
              type="password"
              className="input"
              value={clientSecret}
              onChange={(e) => setClientSecret(e.target.value)}
              placeholder={t.has_gre_credenciales ? "(dejar vacío para no cambiar el actual)" : ""}
            />
          </div>
          <div className="flex justify-end">
            <button type="submit" className="btn-primary" disabled={saving}>
              {saving ? "Guardando…" : "Guardar credenciales GRE"}
            </button>
          </div>
        </form>
      )}
    </section>
  );
}

function PrinterSection() {
  const [device, setDevice] = useState<PrinterDevice | null>(getSavedPrinter());
  const [settings, setSettings] = useState<PrinterSettings>(getPrinterSettings());
  const [busy, setBusy] = useState(false);
  const cap = detectCapability();

  function updateSettings(patch: Partial<PrinterSettings>) {
    const next = { ...settings, ...patch };
    setSettings(next);
    savePrinterSettings(next);
  }

  async function emparejar() {
    setBusy(true);
    try {
      const p = await pickPrinter();
      if (p) {
        setDevice(p);
        toast.success(`Impresora elegida: ${p.name}`);
      }
    } catch (e: any) {
      toast.error(e?.message || "No se pudo emparejar");
    } finally {
      setBusy(false);
    }
  }

  function olvidar() {
    saveSavedPrinter(null);
    setDevice(null);
    toast.success("Impresora olvidada");
  }

  async function prueba() {
    setBusy(true);
    try {
      const p = new EscPos(settings.cols).init()
        .bold(true).size(2, 2).center("PRUEBA DE IMPRESION").size(1, 1).bold(false)
        .feed(1)
        .center("Facturador Self-Hosted")
        .center(new Date().toLocaleString("es-PE"))
        .feed(1)
        .hr()
        .ln("Si ves este ticket completo, la conexion")
        .ln("con la impresora termica esta lista.")
        .hr()
        .feed(3)
        .cut()
        .build();
      await printBytes(p);
      toast.success("Ticket de prueba enviado");
    } catch (e: any) {
      toast.error(e?.message || "Falló la impresión");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="card p-6">
      <h2 className="text-lg font-medium text-slate-900 mb-1">Impresora térmica</h2>
      <p className="text-xs text-slate-500 mb-4">
        Para imprimir tickets de 80mm o 58mm a impresoras térmicas Bluetooth
        (Xprinter, Bixolon, EPSON TM, etc.). El cajero la empareja una vez y
        cada impresión usa la guardada.
      </p>

      <div className={`mb-4 rounded-lg p-3 text-sm border ${
        cap === "unsupported"
          ? "bg-rose-50 border-rose-200 text-rose-800"
          : "bg-slate-50 border-slate-200 text-slate-700"
      }`}>
        {cap === "native" && (
          <>App nativa Android — soporte completo de Bluetooth Classic + BLE.</>
        )}
        {cap === "web-bluetooth" && (
          <>
            Navegador con Web Bluetooth (Chrome / Edge en Android o
            escritorio). En el primer uso vas a tener que confirmar el
            permiso de Bluetooth.
          </>
        )}
        {cap === "unsupported" && (
          <>
            Tu navegador <strong>no soporta Bluetooth</strong> (Safari y
            Firefox lo bloquean). Usá Chrome en Android o instalá la app
            del Facturador desde Play Store.
          </>
        )}
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between rounded-lg border border-slate-200 p-3">
          <div>
            <div className="text-xs uppercase tracking-wide text-slate-500">
              Impresora actual
            </div>
            <div className="font-medium text-slate-900">
              {device ? device.name : "— ninguna emparejada —"}
            </div>
            {device && (
              <div className="font-mono text-xs text-slate-400 mt-0.5">
                {device.id}
              </div>
            )}
          </div>
          {device && (
            <button onClick={olvidar} className="text-xs text-rose-600 hover:underline">
              Olvidar
            </button>
          )}
        </div>

        <div className="flex gap-2">
          <button
            onClick={emparejar}
            className="btn-primary flex-1"
            disabled={busy || cap === "unsupported"}
          >
            {device ? "Cambiar impresora" : "Emparejar impresora"}
          </button>
          <button
            onClick={prueba}
            className="btn-ghost"
            disabled={busy || !device}
          >
            Imprimir prueba
          </button>
        </div>

        <div className="pt-4 mt-2 border-t border-slate-100 space-y-3">
          <label className="flex items-start gap-3 cursor-pointer">
            <input
              type="checkbox"
              className="mt-0.5"
              checked={settings.auto}
              onChange={(e) => updateSettings({ auto: e.target.checked })}
            />
            <span>
              <span className="text-sm font-medium text-slate-900 block">
                Imprimir automáticamente al aceptar
              </span>
              <span className="text-xs text-slate-500">
                Apenas SUNAT confirma el comprobante, el ticket sale solo. El
                cajero no necesita tocar nada más.
              </span>
            </span>
          </label>

          <div>
            <label className="label">Ancho del papel</label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => updateSettings({ cols: 42 })}
                className={`flex-1 rounded-lg border px-3 py-2 text-sm font-medium transition ${
                  settings.cols === 42
                    ? "border-brand-500 bg-brand-50 text-brand-700"
                    : "border-slate-200 bg-white text-slate-700 hover:bg-slate-50"
                }`}
              >
                80 mm (común)
                <div className="text-xs text-slate-500 font-normal">42 caracteres</div>
              </button>
              <button
                type="button"
                onClick={() => updateSettings({ cols: 32 })}
                className={`flex-1 rounded-lg border px-3 py-2 text-sm font-medium transition ${
                  settings.cols === 32
                    ? "border-brand-500 bg-brand-50 text-brand-700"
                    : "border-slate-200 bg-white text-slate-700 hover:bg-slate-50"
                }`}
              >
                58 mm (compacta)
                <div className="text-xs text-slate-500 font-normal">32 caracteres</div>
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

const BASE_API = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

function CertSection({
  tenant: t, esDueno, onUploaded,
}: { tenant: TenantConfig; esDueno: boolean; onUploaded: () => void }) {
  const [file, setFile] = useState<File | null>(null);
  const [passphrase, setPassphrase] = useState("");
  const [uploading, setUploading] = useState(false);

  async function onUpload(e: FormEvent) {
    e.preventDefault();
    if (!file) {
      toast.error("Seleccioná el archivo .p12");
      return;
    }
    if (!passphrase) {
      toast.error("La passphrase es obligatoria");
      return;
    }
    const tokens = JSON.parse(localStorage.getItem("facturador.tokens") || "null");
    if (!tokens?.access) {
      toast.error("Sesión expirada");
      return;
    }
    setUploading(true);
    const fd = new FormData();
    fd.append("file", file);
    fd.append("passphrase", passphrase);
    try {
      const r = await fetch(`${BASE_API}/api/v1/tenant/cert`, {
        method: "POST",
        headers: { Authorization: `Bearer ${tokens.access}` },
        body: fd,
      });
      if (!r.ok) {
        const body = await r.json().catch(() => ({}));
        throw new Error(body?.detalle || body?.error || `HTTP ${r.status}`);
      }
      toast.success("Certificado cargado correctamente");
      setFile(null);
      setPassphrase("");
      onUploaded();
    } catch (err: any) {
      toast.error("No se pudo cargar: " + (err?.message || "error"));
    } finally {
      setUploading(false);
    }
  }

  return (
    <section className="card p-6">
      <h2 className="text-lg font-medium text-slate-900 mb-1">Certificado Digital Tributario</h2>
      <p className="text-xs text-slate-500 mb-4">
        Tu archivo <code>.p12</code> descargado desde Clave SOL. La passphrase se
        cifra con AES-256-GCM antes de guardarse, para que el container pueda
        recargar el certificado al reiniciar sin pedirte la clave de nuevo.
      </p>

      <div className={`mb-4 rounded-lg p-3 text-sm border ${
        t.has_cert
          ? "bg-emerald-50 border-emerald-200 text-emerald-800"
          : "bg-rose-50 border-rose-200 text-rose-800"
      }`}>
        {t.has_cert ? (
          <>✓ Certificado cargado y listo para emitir.</>
        ) : (
          <>⚠ Esta empresa todavía no tiene certificado. Subilo abajo antes de emitir comprobantes.</>
        )}
      </div>

      {esDueno && (
        <form onSubmit={onUpload} className="space-y-4">
          <div>
            <label className="label">Archivo .p12</label>
            <input
              type="file"
              accept=".p12,.pfx,application/x-pkcs12"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
              className="block w-full text-sm text-slate-700 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:bg-brand-50 file:text-brand-700 hover:file:bg-brand-100 cursor-pointer"
            />
            {file && (
              <p className="text-xs text-slate-500 mt-1">
                Seleccionado: {file.name} ({Math.round(file.size / 1024)} KB)
              </p>
            )}
          </div>
          <div>
            <label className="label">Passphrase del .p12</label>
            <input
              type="password"
              className="input"
              value={passphrase}
              onChange={(e) => setPassphrase(e.target.value)}
              placeholder="La que pusiste al generarlo en Clave SOL"
            />
          </div>
          <div className="flex justify-end">
            <button type="submit" className="btn-primary" disabled={uploading || !file}>
              {uploading ? "Subiendo…" : t.has_cert ? "Reemplazar certificado" : "Subir certificado"}
            </button>
          </div>
        </form>
      )}
    </section>
  );
}

