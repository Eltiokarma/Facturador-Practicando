import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { api } from "../api/client";
import { useAuth } from "../auth/AuthContext";

export function NuevaEmpresa() {
  const nav = useNavigate();
  const { reloadTenants, switchTenant } = useAuth();
  const [ruc, setRuc] = useState("");
  const [razon, setRazon] = useState("");
  const [nombreComercial, setNombreComercial] = useState("");
  const [direccion, setDireccion] = useState("");
  const [ubigeo, setUbigeo] = useState("");
  const [saving, setSaving] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!/^\d{11}$/.test(ruc)) {
      toast.error("El RUC debe tener 11 dígitos");
      return;
    }
    if (!razon.trim()) {
      toast.error("La razón social es obligatoria");
      return;
    }
    setSaving(true);
    try {
      const r = await api<{ id: string }>("/api/v1/tenants", {
        body: {
          ruc,
          razon_social: razon,
          nombre_comercial: nombreComercial,
          direccion_fiscal: direccion,
          ubigeo,
        },
      });
      await reloadTenants();
      toast.success("Empresa creada. Cambiando a ella…");
      await switchTenant(r.id);
      // Tras switchTenant la app necesita recargar para tomar el contexto nuevo
      window.location.href = "/configuracion";
    } catch (e: any) {
      toast.error("No se pudo crear: " + (e?.body?.detalle || e?.message || "error"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="max-w-2xl space-y-6">
      <header>
        <h1 className="text-2xl font-semibold text-slate-900">Nueva empresa</h1>
        <p className="text-sm text-slate-500 mt-1">
          Agregá un RUC adicional bajo tu usuario. Vas a quedar como dueño del
          nuevo tenant. Después podés subir su certificado y configurar credenciales SUNAT.
        </p>
      </header>

      <form onSubmit={onSubmit} className="card p-6 space-y-4">
        <div>
          <label className="label">RUC</label>
          <input
            className="input"
            value={ruc}
            maxLength={11}
            onChange={(e) => setRuc(e.target.value.replace(/\D/g, ""))}
            placeholder="20XXXXXXXXX"
            required
          />
        </div>
        <div>
          <label className="label">Razón social</label>
          <input
            className="input"
            value={razon}
            onChange={(e) => setRazon(e.target.value)}
            placeholder="MI EMPRESA SAC"
            required
          />
        </div>
        <div>
          <label className="label">Nombre comercial (opcional)</label>
          <input
            className="input"
            value={nombreComercial}
            onChange={(e) => setNombreComercial(e.target.value)}
          />
        </div>
        <div>
          <label className="label">Dirección fiscal (opcional)</label>
          <input
            className="input"
            value={direccion}
            onChange={(e) => setDireccion(e.target.value)}
          />
        </div>
        <div>
          <label className="label">Ubigeo (opcional, 6 dígitos)</label>
          <input
            className="input max-w-xs"
            value={ubigeo}
            maxLength={6}
            onChange={(e) => setUbigeo(e.target.value.replace(/\D/g, ""))}
          />
        </div>
        <div className="rounded-lg p-3 text-sm bg-slate-100 border border-slate-200 text-slate-700">
          La empresa se crea en modo <strong>beta</strong> por defecto. Antes de
          emitir, vas a tener que subir su certificado .p12 y completar las
          credenciales SOL desde Configuración.
        </div>
        <div className="flex justify-end gap-2">
          <button type="button" className="btn-ghost" onClick={() => nav(-1)}>
            Cancelar
          </button>
          <button type="submit" className="btn-primary" disabled={saving}>
            {saving ? "Creando…" : "Crear empresa"}
          </button>
        </div>
      </form>
    </div>
  );
}
