import { FormEvent, useEffect, useState } from "react";
import { toast } from "sonner";
import { api } from "../api/client";
import { useAuth } from "../auth/AuthContext";

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
        },
      });
      toast.success("Datos guardados. Algunos cambios requieren reiniciar el servicio.");
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

      <div className="rounded-lg p-4 text-sm bg-slate-100 border border-slate-200 text-slate-700">
        <strong>Aviso:</strong> los cambios en los datos del emisor y las credenciales SOL
        se guardan inmediatamente, pero <strong>requieren reiniciar el servicio</strong> para que
        las emisiones futuras los usen. El certificado <code>.p12</code> sigue cargándose desde
        el archivo en <code>./certs/cert.p12</code> al arrancar — subirlo por la UI llega en
        la próxima versión.
      </div>

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
    </div>
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

