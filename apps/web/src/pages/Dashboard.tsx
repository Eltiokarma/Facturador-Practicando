import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { ComprobanteRow } from "../types";

export function Dashboard() {
  const { user } = useAuth();
  const [recientes, setRecientes] = useState<ComprobanteRow[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    api<{ items: ComprobanteRow[] }>("/api/v1/comprobantes?limit=5")
      .then((r) => setRecientes(r.items || []))
      .catch((e) => setErr(e.message || "error"));
  }, []);

  const stats = (recientes || []).reduce(
    (acc, c) => {
      acc.total++;
      if (c.estado === "aceptado" || c.estado === "aceptado_con_obs") acc.ok++;
      else if (c.estado === "rechazado") acc.rej++;
      return acc;
    },
    { total: 0, ok: 0, rej: 0 }
  );

  return (
    <div className="space-y-8">
      <header>
        <h1 className="text-2xl font-semibold text-slate-900">
          Hola, {user?.nombre.split(" ")[0] || "—"}
        </h1>
        <p className="text-slate-500 mt-1 text-sm">
          Acá está el resumen de tu facturación reciente.
        </p>
      </header>

      <section className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Stat label="Comprobantes recientes" value={stats.total} hint="últimos 5" />
        <Stat label="Aceptados por SUNAT" value={stats.ok} accent="ok" />
        <Stat label="Rechazados" value={stats.rej} accent={stats.rej > 0 ? "err" : "info"} />
      </section>

      <section className="card p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-medium text-slate-900">Últimos comprobantes</h2>
          <Link to="/comprobantes" className="text-sm text-brand-600 hover:underline">
            Ver todos →
          </Link>
        </div>
        {err && <p className="text-rose-600 text-sm">{err}</p>}
        {!err && recientes === null && (
          <p className="text-slate-500 text-sm">Cargando…</p>
        )}
        {!err && recientes && recientes.length === 0 && (
          <div className="text-center py-10">
            <p className="text-slate-500 text-sm">Todavía no emitiste comprobantes.</p>
            <Link to="/emitir" className="btn-primary mt-4 inline-flex">
              Emitir el primero
            </Link>
          </div>
        )}
        {recientes && recientes.length > 0 && (
          <ul className="divide-y divide-slate-100">
            {recientes.map((c) => (
              <li key={c.id} className="py-3 flex items-center justify-between">
                <div>
                  <Link
                    to={`/comprobantes/${c.id}`}
                    className="text-sm font-medium text-slate-900 hover:text-brand-700"
                  >
                    {labelTipo(c.tipo)} {c.serie}-{c.correlativo}
                  </Link>
                  <div className="text-xs text-slate-500">
                    {c.receptor_razon} · {c.fecha_emision}
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-sm text-slate-700 tabular-nums">
                    S/ {c.total.toFixed(2)}
                  </span>
                  <EstadoBadge estado={c.estado} />
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function Stat({
  label, value, hint, accent,
}: { label: string; value: number; hint?: string; accent?: "ok" | "err" | "info" }) {
  const accentCls =
    accent === "ok" ? "text-emerald-600"
      : accent === "err" ? "text-rose-600"
      : "text-slate-900";
  return (
    <div className="card p-5">
      <div className="text-xs uppercase tracking-wide text-slate-500">{label}</div>
      <div className={`mt-2 text-3xl font-semibold ${accentCls}`}>{value}</div>
      {hint && <div className="text-xs text-slate-400 mt-1">{hint}</div>}
    </div>
  );
}

export function EstadoBadge({ estado }: { estado: string }) {
  switch (estado) {
    case "aceptado":
      return <span className="badge-ok">Aceptado</span>;
    case "aceptado_con_obs":
      return <span className="badge-warn">Con observaciones</span>;
    case "rechazado":
      return <span className="badge-err">Rechazado</span>;
    case "error":
      return <span className="badge-err">Error</span>;
    case "pendiente":
    case "enviando":
      return <span className="badge-info">Pendiente</span>;
    default:
      return <span className="badge-info">{estado}</span>;
  }
}

export function labelTipo(t: string): string {
  switch (t) {
    case "01": return "Factura";
    case "03": return "Boleta";
    case "07": return "N. Crédito";
    case "08": return "N. Débito";
    default: return t;
  }
}
