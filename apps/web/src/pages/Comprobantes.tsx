import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { api } from "../api/client";
import { ComprobanteRow } from "../types";
import { EstadoBadge, labelTipo } from "./Dashboard";

const FILTROS: { value: string; label: string }[] = [
  { value: "", label: "Todos" },
  { value: "aceptado", label: "Aceptados" },
  { value: "aceptado_con_obs", label: "Con observaciones" },
  { value: "rechazado", label: "Rechazados" },
  { value: "error", label: "Error" },
];

export function Comprobantes() {
  const [params, setParams] = useSearchParams();
  const filtro = params.get("estado") || "";
  function setFiltro(v: string) {
    if (v) params.set("estado", v);
    else params.delete("estado");
    setParams(params, { replace: true });
  }
  const [items, setItems] = useState<ComprobanteRow[] | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    const params = new URLSearchParams({ limit: "100" });
    if (filtro) params.set("estado", filtro);
    api<{ items: ComprobanteRow[] }>(`/api/v1/comprobantes?${params}`)
      .then((r) => setItems(r.items || []))
      .catch((e) => {
        toast.error("No se pudo cargar la lista");
        setItems([]);
        console.error(e);
      })
      .finally(() => setLoading(false));
  }, [filtro]);

  return (
    <div className="space-y-6">
      <header className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">Comprobantes</h1>
          <p className="text-sm text-slate-500 mt-1">
            Tu historial completo, con el estado que reportó SUNAT.
          </p>
        </div>
        <Link to="/emitir" className="btn-primary">
          + Nuevo
        </Link>
      </header>

      <div className="flex flex-wrap gap-2">
        {FILTROS.map((f) => (
          <button
            key={f.value}
            onClick={() => setFiltro(f.value)}
            className={`px-3 py-1.5 rounded-full text-sm font-medium border transition ${
              filtro === f.value
                ? "bg-brand-600 text-white border-brand-600"
                : "bg-white text-slate-700 border-slate-200 hover:bg-slate-50"
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      <div className="card overflow-hidden">
        {loading && (
          <div className="px-6 py-10 text-center text-slate-500 text-sm">
            Cargando…
          </div>
        )}
        {!loading && items && items.length === 0 && (
          <div className="px-6 py-10 text-center text-slate-500 text-sm">
            No hay comprobantes para este filtro.
          </div>
        )}
        {!loading && items && items.length > 0 && (
          <table className="min-w-full divide-y divide-slate-100 text-sm">
            <thead className="bg-slate-50">
              <tr>
                <Th>Tipo</Th>
                <Th>Número</Th>
                <Th>Fecha</Th>
                <Th>Cliente</Th>
                <Th className="text-right">Total</Th>
                <Th>Estado</Th>
                <Th />
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {items.map((c) => (
                <tr key={c.id} className="hover:bg-slate-50">
                  <Td>{labelTipo(c.tipo)}</Td>
                  <Td className="font-mono">
                    {c.serie}-{c.correlativo}
                  </Td>
                  <Td className="text-slate-600">{c.fecha_emision}</Td>
                  <Td>
                    <div className="text-slate-900">{c.receptor_razon}</div>
                    <div className="text-xs text-slate-500">{c.receptor_doc}</div>
                  </Td>
                  <Td className="text-right tabular-nums">
                    {c.moneda} {c.total.toFixed(2)}
                  </Td>
                  <Td><EstadoBadge estado={c.estado} /></Td>
                  <Td className="text-right">
                    <Link
                      to={`/comprobantes/${c.id}`}
                      className="text-brand-600 hover:underline text-sm"
                    >
                      Ver →
                    </Link>
                  </Td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

function Th({ children, className = "" }: any) {
  return (
    <th className={`px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 ${className}`}>
      {children}
    </th>
  );
}
function Td({ children, className = "" }: any) {
  return <td className={`px-4 py-3 align-top ${className}`}>{children}</td>;
}
