import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { useAuth } from "../auth/AuthContext";

export function TenantSwitcher() {
  const { user, tenants, switchTenant } = useAuth();
  const nav = useNavigate();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  if (!user) return null;
  const current = tenants.find((t) => t.id === user.tenant_id);

  async function pick(tenantID: string) {
    if (tenantID === user!.tenant_id) {
      setOpen(false);
      return;
    }
    try {
      await switchTenant(tenantID);
      toast.success("Empresa cambiada");
      setOpen(false);
      // Recargar para que las pantallas tomen el nuevo tenant
      window.location.reload();
    } catch (e: any) {
      toast.error("No se pudo cambiar: " + (e?.message || "error"));
    }
  }

  function crear() {
    setOpen(false);
    nav("/empresas/nueva");
  }

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-left hover:bg-slate-50"
      >
        <div className="min-w-0">
          <div className="text-xs uppercase tracking-wide text-slate-500">Empresa</div>
          <div className="text-sm font-medium text-slate-900 truncate">
            {current?.razon_social || "—"}
          </div>
          <div className="text-xs text-slate-500 truncate">
            RUC {current?.ruc || "—"}
            {current?.sunat_mode === "prod" ? (
              <span className="ml-2 text-amber-700">· prod</span>
            ) : (
              <span className="ml-2 text-emerald-700">· beta</span>
            )}
          </div>
        </div>
        <span className="text-slate-400">⌄</span>
      </button>
      {open && (
        <div className="absolute z-20 left-0 right-0 mt-1 bg-white border border-slate-200 rounded-lg shadow-lg max-h-80 overflow-auto">
          {tenants.map((t) => (
            <button
              key={t.id}
              onClick={() => pick(t.id)}
              className={`block w-full text-left px-3 py-2 text-sm hover:bg-slate-50 ${
                t.id === user.tenant_id ? "bg-brand-50" : ""
              }`}
            >
              <div className="font-medium text-slate-900 truncate">{t.razon_social}</div>
              <div className="text-xs text-slate-500">
                RUC {t.ruc}
                {!t.has_cert && (
                  <span className="ml-2 text-rose-600">· sin cert</span>
                )}
              </div>
            </button>
          ))}
          <div className="border-t border-slate-100">
            <button
              onClick={crear}
              className="block w-full text-left px-3 py-2 text-sm text-brand-700 hover:bg-brand-50"
            >
              + Nueva empresa
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
