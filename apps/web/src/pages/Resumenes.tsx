import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { api } from "../api/client";

const HOY = new Date().toISOString().slice(0, 10);

type BoletaPendiente = {
  id: string;
  serie: string;
  correlativo: number;
  receptor_doc: string;
  receptor_razon: string;
  moneda: string;
  total: number;
  igv: number;
};

type ResumenRow = {
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
  created_at: string;
};

function estadoBadge(estado: string) {
  switch (estado) {
    case "aceptado":
      return <span className="badge-ok">Aceptado</span>;
    case "aceptado_con_obs":
      return <span className="badge-warn">Con observaciones</span>;
    case "rechazado":
    case "error":
      return <span className="badge-err">{estado}</span>;
    default:
      return <span className="badge-info">{estado}</span>;
  }
}

export function Resumenes() {
  const nav = useNavigate();
  const [fecha, setFecha] = useState(HOY);
  const [pendientes, setPendientes] = useState<BoletaPendiente[] | null>(null);
  const [resumenes, setResumenes] = useState<ResumenRow[] | null>(null);
  const [enviando, setEnviando] = useState(false);

  async function cargar() {
    try {
      const [p, r] = await Promise.all([
        api<{ items: BoletaPendiente[] }>(`/api/v1/resumenes/pendientes?fecha=${fecha}`),
        api<{ items: ResumenRow[] }>(`/api/v1/resumenes`),
      ]);
      setPendientes(p.items || []);
      setResumenes(r.items || []);
    } catch (e: any) {
      toast.error("No se pudo cargar: " + e.message);
    }
  }

  useEffect(() => {
    cargar();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fecha]);

  const total = (pendientes || []).reduce((acc, b) => acc + b.total, 0);
  const totalIGV = (pendientes || []).reduce((acc, b) => acc + b.igv, 0);

  async function enviarResumen() {
    if (!pendientes || pendientes.length === 0) return;
    if (!confirm(`¿Enviar resumen de ${pendientes.length} boleta(s) del ${fecha} a SUNAT?`)) return;
    setEnviando(true);
    try {
      const r = await api<ResumenRow>("/api/v1/resumenes", {
        body: { fecha },
      });
      toast.success(`Resumen ${r.serie}-${r.correlativo} encolado, esperando respuesta de SUNAT…`);
      nav(`/resumenes/${r.id}`);
    } catch (e: any) {
      toast.error("No se pudo enviar: " + (e?.body?.detalle || e.message));
    } finally {
      setEnviando(false);
    }
  }

  return (
    <div className="space-y-8">
      <header>
        <h1 className="text-2xl font-semibold text-slate-900">Resumen diario</h1>
        <p className="text-sm text-slate-500 mt-1">
          Las boletas se reportan a SUNAT en un resumen consolidado, no una por una.
          El plazo es 7 días desde su emisión.
        </p>
      </header>

      <section className="card p-6 space-y-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
          Boletas pendientes de un día
        </h2>
        <div className="flex items-center gap-3">
          <label className="text-sm text-slate-600">Fecha:</label>
          <input
            type="date"
            className="input max-w-xs"
            value={fecha}
            onChange={(e) => setFecha(e.target.value)}
          />
          <div className="flex-1" />
          <button
            className="btn-primary"
            onClick={enviarResumen}
            disabled={enviando || !pendientes || pendientes.length === 0}
          >
            {enviando ? "Enviando…" : `Enviar resumen (${pendientes?.length || 0})`}
          </button>
        </div>

        {pendientes === null && (
          <div className="text-sm text-slate-500 text-center py-6">Cargando…</div>
        )}
        {pendientes && pendientes.length === 0 && (
          <div className="text-sm text-slate-500 text-center py-6 bg-slate-50 rounded-lg">
            No hay boletas pendientes para esa fecha. (Solo se listan las que ya
            fueron aceptadas individualmente por SUNAT y no están en otro resumen.)
          </div>
        )}
        {pendientes && pendientes.length > 0 && (
          <>
            <div className="overflow-hidden border border-slate-200 rounded-lg">
              <table className="min-w-full divide-y divide-slate-100 text-sm">
                <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
                  <tr>
                    <th className="px-4 py-2 text-left">Boleta</th>
                    <th className="px-4 py-2 text-left">Cliente</th>
                    <th className="px-4 py-2 text-right">IGV</th>
                    <th className="px-4 py-2 text-right">Total</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {pendientes.map((b) => (
                    <tr key={b.id}>
                      <td className="px-4 py-2 font-mono">{b.serie}-{b.correlativo}</td>
                      <td className="px-4 py-2">
                        <div>{b.receptor_razon}</div>
                        <div className="text-xs text-slate-500">{b.receptor_doc}</div>
                      </td>
                      <td className="px-4 py-2 text-right tabular-nums">{b.igv.toFixed(2)}</td>
                      <td className="px-4 py-2 text-right tabular-nums">{b.moneda} {b.total.toFixed(2)}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot className="bg-slate-50 text-sm">
                  <tr>
                    <td colSpan={2} className="px-4 py-2 font-medium text-slate-600">
                      {pendientes.length} boleta(s)
                    </td>
                    <td className="px-4 py-2 text-right tabular-nums font-medium">{totalIGV.toFixed(2)}</td>
                    <td className="px-4 py-2 text-right tabular-nums font-medium">
                      PEN {total.toFixed(2)}
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </>
        )}
      </section>

      <section className="card p-6 space-y-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
          Historial de resúmenes
        </h2>
        {!resumenes && <p className="text-slate-500 text-sm">Cargando…</p>}
        {resumenes && resumenes.length === 0 && (
          <p className="text-slate-500 text-sm">Todavía no enviaste ningún resumen.</p>
        )}
        {resumenes && resumenes.length > 0 && (
          <table className="min-w-full divide-y divide-slate-100 text-sm">
            <thead className="text-xs uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-2 py-2 text-left">Resumen</th>
                <th className="px-2 py-2 text-left">Día reportado</th>
                <th className="px-2 py-2 text-center">Boletas</th>
                <th className="px-2 py-2">Estado</th>
                <th className="px-2 py-2" />
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {resumenes.map((r) => (
                <tr key={r.id} className="hover:bg-slate-50">
                  <td className="px-2 py-2 font-mono">{r.serie}-{r.correlativo}</td>
                  <td className="px-2 py-2 text-slate-600">{r.fecha_referencia}</td>
                  <td className="px-2 py-2 text-center tabular-nums">{r.cantidad_comprobantes}</td>
                  <td className="px-2 py-2">{estadoBadge(r.estado)}</td>
                  <td className="px-2 py-2 text-right">
                    <Link to={`/resumenes/${r.id}`} className="text-brand-600 hover:underline">
                      Ver →
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  );
}
