import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";
import { api } from "../api/client";

type Anulacion = {
  id: string;
  serie: string;
  correlativo: number;
  fecha_referencia: string;
  fecha_emision: string;
  estado: string;
  ticket?: string;
  sunat_codigo?: string;
  sunat_mensaje?: string;
  created_at: string;
};

type ComprobanteAnular = {
  id: string;
  tipo: string;
  serie: string;
  correlativo: number;
  receptor_razon: string;
  total: number;
  moneda: string;
  motivo: string;
};

type Detalle = Anulacion & { comprobantes: ComprobanteAnular[] };

const EN_PROCESO = new Set(["pendiente", "enviando", "consultando", "error"]);

function estadoBadge(estado: string) {
  switch (estado) {
    case "aceptado": return <span className="badge-ok">Aceptado</span>;
    case "aceptado_con_obs": return <span className="badge-warn">Con observaciones</span>;
    case "rechazado":
    case "error": return <span className="badge-err">{estado}</span>;
    default: return <span className="badge-info">{estado}</span>;
  }
}

export function Anulaciones() {
  const [items, setItems] = useState<Anulacion[] | null>(null);
  useEffect(() => {
    api<{ items: Anulacion[] }>("/api/v1/anulaciones")
      .then((r) => setItems(r.items || []))
      .catch(() => toast.error("No se pudo cargar"));
  }, []);

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold text-slate-900">Anulaciones</h1>
        <p className="text-sm text-slate-500 mt-1">
          Comunicaciones de baja enviadas a SUNAT. Cada una anula uno o más
          comprobantes ya aceptados (dentro de los 7 días siguientes a la emisión).
        </p>
      </header>

      <div className="card overflow-hidden">
        {!items && <div className="p-10 text-center text-slate-500 text-sm">Cargando…</div>}
        {items && items.length === 0 && (
          <div className="p-10 text-center text-slate-500 text-sm">
            No has emitido ninguna comunicación de baja todavía. Para anular un
            comprobante, andá a su pantalla de detalle y usá "Anular".
          </div>
        )}
        {items && items.length > 0 && (
          <table className="min-w-full divide-y divide-slate-100 text-sm">
            <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-4 py-2 text-left">Número</th>
                <th className="px-4 py-2 text-left">Día anulado</th>
                <th className="px-4 py-2 text-left">Fecha envío</th>
                <th className="px-4 py-2">Estado</th>
                <th className="px-4 py-2" />
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {items.map((a) => (
                <tr key={a.id} className="hover:bg-slate-50">
                  <td className="px-4 py-2 font-mono">{a.serie}-{a.correlativo}</td>
                  <td className="px-4 py-2 text-slate-600">{a.fecha_referencia}</td>
                  <td className="px-4 py-2 text-slate-600">{a.fecha_emision}</td>
                  <td className="px-4 py-2">{estadoBadge(a.estado)}</td>
                  <td className="px-4 py-2 text-right">
                    <Link to={`/anulaciones/${a.id}`} className="text-brand-600 hover:underline">
                      Ver →
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

export function AnulacionDetalle() {
  const { id } = useParams<{ id: string }>();
  const nav = useNavigate();
  const [d, setD] = useState<Detalle | null>(null);
  const lastEstado = useRef<string | null>(null);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    async function fetchOne() {
      try {
        const data = await api<Detalle>(`/api/v1/anulaciones/${id}`);
        if (cancelled) return;
        setD(data);
        if (lastEstado.current && EN_PROCESO.has(lastEstado.current)) {
          if (data.estado === "aceptado") toast.success("SUNAT aceptó la anulación");
          else if (data.estado === "rechazado") toast.error("SUNAT rechazó la anulación");
        }
        lastEstado.current = data.estado;
      } catch {
        if (!cancelled) {
          toast.error("No encontrado");
          nav("/anulaciones", { replace: true });
        }
      }
    }
    fetchOne();
    const t = window.setInterval(() => {
      if (cancelled) return;
      if (lastEstado.current && EN_PROCESO.has(lastEstado.current)) fetchOne();
    }, 3000);
    return () => {
      cancelled = true;
      window.clearInterval(t);
    };
  }, [id, nav]);

  if (!d) return <div className="text-slate-500 text-sm">Cargando…</div>;

  return (
    <div className="space-y-6">
      <header>
        <Link to="/anulaciones" className="text-sm text-brand-600 hover:underline">← Volver</Link>
        <h1 className="text-2xl font-semibold text-slate-900 mt-1">
          Anulación {d.serie}-{d.correlativo}
        </h1>
        <div className="text-sm text-slate-500 mt-1">
          Sobre comprobantes del {d.fecha_referencia} · Enviada el {d.fecha_emision}
        </div>
      </header>

      {EN_PROCESO.has(d.estado) && (
        <div className="rounded-lg p-4 text-sm bg-slate-100 border border-slate-200 text-slate-700 flex items-center gap-3">
          <span className="inline-block h-2 w-2 rounded-full bg-amber-400 animate-pulse" />
          <div>
            <div className="font-medium">
              {d.estado === "consultando"
                ? "Consultando ticket de anulación…"
                : d.estado === "enviando" ? "Enviando a SUNAT…"
                : d.estado === "error" ? "Reintentando…"
                : "Pendiente de envío"}
            </div>
            <div className="text-xs text-slate-500 mt-0.5">
              SUNAT puede tardar minutos. Esto se actualiza solo.
            </div>
          </div>
        </div>
      )}

      {!EN_PROCESO.has(d.estado) && (d.sunat_codigo || d.sunat_mensaje) && (
        <div className={`rounded-lg p-4 text-sm border ${
          d.estado === "rechazado"
            ? "bg-rose-50 text-rose-800 border-rose-200"
            : d.estado === "aceptado_con_obs"
            ? "bg-amber-50 text-amber-800 border-amber-200"
            : "bg-emerald-50 text-emerald-800 border-emerald-200"
        }`}>
          <div className="font-medium">SUNAT respondió:</div>
          <div className="mt-1">
            {d.sunat_codigo && <span className="font-mono mr-2">{d.sunat_codigo}</span>}
            {d.sunat_mensaje}
          </div>
        </div>
      )}

      <section className="card p-6">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 mb-4">
          Comprobantes anulados
        </h2>
        <table className="min-w-full divide-y divide-slate-100 text-sm">
          <thead className="text-xs uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-2 py-2 text-left">Comprobante</th>
              <th className="px-2 py-2 text-left">Cliente</th>
              <th className="px-2 py-2 text-right">Total</th>
              <th className="px-2 py-2 text-left">Motivo</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {d.comprobantes.map((c) => (
              <tr key={c.id}>
                <td className="px-2 py-2 font-mono">
                  <Link to={`/comprobantes/${c.id}`} className="text-brand-600 hover:underline">
                    {c.serie}-{c.correlativo}
                  </Link>
                </td>
                <td className="px-2 py-2">{c.receptor_razon}</td>
                <td className="px-2 py-2 text-right tabular-nums">
                  {c.moneda} {c.total.toFixed(2)}
                </td>
                <td className="px-2 py-2 text-slate-600">{c.motivo}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {d.ticket && (
        <section className="card p-4 text-xs text-slate-500">
          Ticket SUNAT: <span className="font-mono text-slate-700">{d.ticket}</span>
        </section>
      )}
    </div>
  );
}
