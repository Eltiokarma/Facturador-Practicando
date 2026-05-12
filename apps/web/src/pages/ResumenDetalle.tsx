import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";
import { api } from "../api/client";

type Boleta = {
  id: string;
  serie: string;
  correlativo: number;
  receptor_doc: string;
  receptor_razon: string;
  moneda: string;
  total: number;
  igv: number;
};

type ResumenDetalleT = {
  id: string;
  serie: string;
  correlativo: number;
  fecha_referencia: string;
  fecha_emision: string;
  estado: string;
  ticket?: string;
  sunat_codigo?: string;
  sunat_mensaje?: string;
  cantidad_comprobantes: number;
  boletas: Boleta[];
};

const EN_PROCESO = new Set(["pendiente", "enviando", "consultando", "error"]);

export function ResumenDetalle() {
  const { id } = useParams<{ id: string }>();
  const nav = useNavigate();
  const [r, setR] = useState<ResumenDetalleT | null>(null);
  const [loading, setLoading] = useState(true);
  const lastEstado = useRef<string | null>(null);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;

    async function fetchOne() {
      try {
        const data = await api<ResumenDetalleT>(`/api/v1/resumenes/${id}`);
        if (cancelled) return;
        setR(data);
        if (lastEstado.current && EN_PROCESO.has(lastEstado.current)) {
          if (data.estado === "aceptado") toast.success(`SUNAT aceptó ${data.serie}-${data.correlativo}`);
          else if (data.estado === "aceptado_con_obs") toast.warning("Aceptado con observaciones");
          else if (data.estado === "rechazado") toast.error("SUNAT rechazó el resumen");
        }
        lastEstado.current = data.estado;
      } catch {
        if (!cancelled) {
          toast.error("No encontrado");
          nav("/resumenes", { replace: true });
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    fetchOne();
    const tid = window.setInterval(() => {
      if (cancelled) return;
      if (lastEstado.current && EN_PROCESO.has(lastEstado.current)) fetchOne();
    }, 3000);
    return () => {
      cancelled = true;
      window.clearInterval(tid);
    };
  }, [id, nav]);

  if (loading || !r) return <div className="text-slate-500 text-sm">Cargando…</div>;

  return (
    <div className="space-y-6">
      <header>
        <Link to="/resumenes" className="text-sm text-brand-600 hover:underline">← Volver</Link>
        <h1 className="text-2xl font-semibold text-slate-900 mt-1">
          Resumen {r.serie}-{r.correlativo}
        </h1>
        <div className="text-sm text-slate-500 mt-1">
          Boletas del {r.fecha_referencia} · Enviado el {r.fecha_emision}
        </div>
      </header>

      {EN_PROCESO.has(r.estado) && (
        <div className="rounded-lg p-4 text-sm bg-slate-100 text-slate-700 border border-slate-200 flex items-center gap-3">
          <span className="inline-block h-2 w-2 rounded-full bg-amber-400 animate-pulse" />
          <div>
            <div className="font-medium">
              {r.estado === "consultando"
                ? `Consultando ticket ${r.ticket?.slice(0, 12)}…`
                : r.estado === "enviando"
                ? "Enviando a SUNAT…"
                : r.estado === "error"
                ? "Reintentando…"
                : "Pendiente de envío"}
            </div>
            <div className="text-xs text-slate-500 mt-0.5">
              Los resúmenes funcionan con ticket: SUNAT puede tardar de 30 s a varios
              minutos en procesar. Esto se actualiza solo.
            </div>
          </div>
        </div>
      )}

      {!EN_PROCESO.has(r.estado) && (r.sunat_codigo || r.sunat_mensaje) && (
        <div className={`rounded-lg p-4 text-sm border ${
          r.estado === "rechazado"
            ? "bg-rose-50 text-rose-800 border-rose-200"
            : r.estado === "aceptado_con_obs"
            ? "bg-amber-50 text-amber-800 border-amber-200"
            : "bg-emerald-50 text-emerald-800 border-emerald-200"
        }`}>
          <div className="font-medium">SUNAT respondió:</div>
          <div className="mt-1">
            {r.sunat_codigo && <span className="font-mono mr-2">{r.sunat_codigo}</span>}
            {r.sunat_mensaje}
          </div>
        </div>
      )}

      <section className="card p-6">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 mb-4">
          Boletas incluidas ({r.boletas.length})
        </h2>
        <table className="min-w-full divide-y divide-slate-100 text-sm">
          <thead className="text-xs uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-2 py-2 text-left">Boleta</th>
              <th className="px-2 py-2 text-left">Cliente</th>
              <th className="px-2 py-2 text-right">IGV</th>
              <th className="px-2 py-2 text-right">Total</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {r.boletas.map((b) => (
              <tr key={b.id}>
                <td className="px-2 py-2 font-mono">
                  <Link className="text-brand-600 hover:underline" to={`/comprobantes/${b.id}`}>
                    {b.serie}-{b.correlativo}
                  </Link>
                </td>
                <td className="px-2 py-2">
                  <div>{b.receptor_razon}</div>
                  <div className="text-xs text-slate-500">{b.receptor_doc}</div>
                </td>
                <td className="px-2 py-2 text-right tabular-nums">{b.igv.toFixed(2)}</td>
                <td className="px-2 py-2 text-right tabular-nums">{b.moneda} {b.total.toFixed(2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {r.ticket && (
        <section className="card p-4 text-xs text-slate-500">
          Ticket SUNAT: <span className="font-mono text-slate-700">{r.ticket}</span>
        </section>
      )}
    </div>
  );
}
