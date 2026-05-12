import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";
import { api, apiBlob } from "../api/client";
import { ComprobanteDetalleT } from "../types";
import { EstadoBadge, labelTipo } from "./Dashboard";

export function ComprobanteDetalle() {
  const { id } = useParams<{ id: string }>();
  const nav = useNavigate();
  const [c, setC] = useState<ComprobanteDetalleT | null>(null);
  const [loading, setLoading] = useState(true);
  const [showPayload, setShowPayload] = useState(false);

  useEffect(() => {
    if (!id) return;
    api<ComprobanteDetalleT>(`/api/v1/comprobantes/${id}`)
      .then(setC)
      .catch(() => {
        toast.error("No encontrado");
        nav("/comprobantes", { replace: true });
      })
      .finally(() => setLoading(false));
  }, [id, nav]);

  async function descargar(kind: "xml" | "cdr") {
    if (!id) return;
    try {
      const blob = await apiBlob(`/api/v1/comprobantes/${id}/${kind}`);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = kind === "xml" ? `${c?.serie}-${c?.correlativo}.xml` : `${c?.serie}-${c?.correlativo}.cdr.zip`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      toast.error(`No hay ${kind.toUpperCase()} disponible`);
    }
  }

  if (loading) {
    return <div className="text-slate-500 text-sm">Cargando…</div>;
  }
  if (!c) return null;

  const items: any[] = c.payload?.Items || c.payload?.items || [];

  return (
    <div className="space-y-6">
      <header className="flex items-start justify-between gap-4">
        <div>
          <Link to="/comprobantes" className="text-sm text-brand-600 hover:underline">
            ← Volver
          </Link>
          <h1 className="text-2xl font-semibold text-slate-900 mt-1">
            {labelTipo(c.tipo)} {c.serie}-{c.correlativo}
          </h1>
          <div className="mt-1 text-sm text-slate-500">
            Emitido el {c.fecha_emision}
          </div>
        </div>
        <EstadoBadge estado={c.estado} />
      </header>

      {(c.sunat_codigo || c.sunat_mensaje) && (
        <div className={`rounded-lg p-4 text-sm ${
          c.estado === "rechazado" || c.estado === "error"
            ? "bg-rose-50 text-rose-800 border border-rose-200"
            : c.estado === "aceptado_con_obs"
            ? "bg-amber-50 text-amber-800 border border-amber-200"
            : "bg-emerald-50 text-emerald-800 border border-emerald-200"
        }`}>
          <div className="font-medium">SUNAT respondió:</div>
          <div className="mt-1">
            {c.sunat_codigo && <span className="font-mono mr-2">{c.sunat_codigo}</span>}
            {c.sunat_mensaje}
          </div>
        </div>
      )}

      <section className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="card p-6 lg:col-span-2">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 mb-4">
            Receptor
          </h2>
          <dl className="space-y-2 text-sm">
            <Row label="Documento" value={`${c.receptor_doc}`} />
            <Row label="Razón / Nombre" value={c.receptor_razon} />
          </dl>

          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 mt-8 mb-4">
            Ítems
          </h2>
          {items.length === 0 ? (
            <p className="text-slate-500 text-sm">Sin detalle disponible.</p>
          ) : (
            <table className="min-w-full text-sm">
              <thead className="text-xs uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="text-left py-2 pr-2">Descripción</th>
                  <th className="text-right py-2 px-2">Cant.</th>
                  <th className="text-right py-2 px-2">V. Unit</th>
                  <th className="text-right py-2 pl-2">Total</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {items.map((it: any, i: number) => (
                  <tr key={i}>
                    <td className="py-2 pr-2">{it.descripcion}</td>
                    <td className="py-2 px-2 text-right tabular-nums">{it.cantidad}</td>
                    <td className="py-2 px-2 text-right tabular-nums">{(+it.valor_unitario || 0).toFixed(2)}</td>
                    <td className="py-2 pl-2 text-right tabular-nums">{(+it.total || 0).toFixed(2)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        <div className="card p-6">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 mb-4">
            Totales
          </h2>
          <dl className="space-y-2 text-sm">
            <Row label="Gravado" value={`${c.moneda} ${c.gravado.toFixed(2)}`} />
            <Row label="Exonerado" value={`${c.moneda} ${c.exonerado.toFixed(2)}`} />
            <Row label="Inafecto" value={`${c.moneda} ${c.inafecto.toFixed(2)}`} />
            <Row label="IGV" value={`${c.moneda} ${c.igv.toFixed(2)}`} />
          </dl>
          <div className="border-t border-slate-200 mt-4 pt-4 flex justify-between items-baseline">
            <span className="text-sm text-slate-600">Total</span>
            <span className="text-xl font-semibold tabular-nums">
              {c.moneda} {c.total.toFixed(2)}
            </span>
          </div>

          {c.hash_cpe && (
            <div className="mt-6 text-xs">
              <div className="uppercase tracking-wide text-slate-500 font-medium">Hash CPE</div>
              <div className="mt-1 font-mono text-slate-700 break-all">{c.hash_cpe}</div>
            </div>
          )}

          <div className="mt-6 space-y-2">
            <button className="btn-ghost w-full" onClick={() => descargar("xml")}>
              Descargar XML firmado
            </button>
            <button className="btn-ghost w-full" onClick={() => descargar("cdr")}>
              Descargar CDR (.zip)
            </button>
          </div>
        </div>
      </section>

      <section className="card p-6">
        <button
          className="text-sm text-slate-600 hover:text-slate-900"
          onClick={() => setShowPayload((v) => !v)}
        >
          {showPayload ? "Ocultar" : "Ver"} payload completo (JSON)
        </button>
        {showPayload && (
          <pre className="mt-4 text-xs bg-slate-50 border border-slate-200 rounded-lg p-4 overflow-auto max-h-96">
            {JSON.stringify(c.payload, null, 2)}
          </pre>
        )}
      </section>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between">
      <dt className="text-slate-500">{label}</dt>
      <dd className="text-slate-900 tabular-nums">{value}</dd>
    </div>
  );
}
