import { FormEvent, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { api, ApiError } from "../api/client";
import { ComprobanteDetalleT, Item } from "../types";

// Catálogo SUNAT 09: motivos de Nota de Crédito (subset)
const MOTIVOS_NC = [
  { value: "01", label: "01 - Anulación de la operación" },
  { value: "02", label: "02 - Anulación por error en el RUC" },
  { value: "03", label: "03 - Corrección por error en la descripción" },
  { value: "04", label: "04 - Descuento global" },
  { value: "05", label: "05 - Descuento por ítem" },
  { value: "06", label: "06 - Devolución total" },
  { value: "07", label: "07 - Devolución por ítem" },
  { value: "08", label: "08 - Bonificación" },
  { value: "09", label: "09 - Disminución en el valor" },
  { value: "10", label: "10 - Otros conceptos" },
  { value: "13", label: "13 - Ajustes operaciones de exportación" },
];

// Catálogo SUNAT 10: motivos de Nota de Débito
const MOTIVOS_ND = [
  { value: "01", label: "01 - Intereses por mora" },
  { value: "02", label: "02 - Aumento en el valor" },
  { value: "03", label: "03 - Penalidades / otros conceptos" },
  { value: "10", label: "10 - Ajustes operaciones de exportación" },
  { value: "11", label: "11 - Ajustes afectos al IVAP" },
];

function calcular(items: Item[]) {
  let gravado = 0, exonerado = 0, inafecto = 0, igv = 0;
  for (const it of items) {
    const sub = round2((+it.cantidad || 0) * (+it.valor_unitario || 0));
    switch (it.afectacion_igv) {
      case "10":
        gravado += sub;
        igv += round2(sub * 0.18);
        break;
      case "20":
        exonerado += sub;
        break;
      case "30":
        inafecto += sub;
        break;
    }
  }
  return {
    gravado: round2(gravado),
    exonerado: round2(exonerado),
    inafecto: round2(inafecto),
    igv: round2(igv),
    total: round2(gravado + exonerado + inafecto + igv),
  };
}
function round2(v: number) { return Math.round(v * 100) / 100; }

export function NuevaNota() {
  const nav = useNavigate();
  const { tipo } = useParams<{ tipo: "credito" | "debito" }>();
  const [params] = useSearchParams();
  const refId = params.get("ref");

  const esCredito = tipo === "credito";
  const tipoCodigo = esCredito ? "07" : "08";
  const motivos = esCredito ? MOTIVOS_NC : MOTIVOS_ND;

  const [ref, setRef] = useState<ComprobanteDetalleT | null>(null);
  const [serie, setSerie] = useState("");
  const [motivo, setMotivo] = useState(motivos[0].value);
  const [descripcion, setDescripcion] = useState("");
  const [items, setItems] = useState<Item[]>([]);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!refId) {
      toast.error("Falta el comprobante de referencia");
      nav("/comprobantes", { replace: true });
      return;
    }
    api<ComprobanteDetalleT>(`/api/v1/comprobantes/${refId}`)
      .then((c) => {
        if (!(c.estado === "aceptado" || c.estado === "aceptado_con_obs")) {
          toast.error("Solo se pueden emitir notas sobre comprobantes aceptados");
          nav(`/comprobantes/${c.id}`, { replace: true });
          return;
        }
        setRef(c);
        // Sugerir serie con el mismo prefijo. Para NC/ND la serie típica empieza con F o B (matching el referenciado).
        setSerie(c.serie);
        // Copiar los items del comprobante original como base.
        const its: Item[] = (c.payload?.items || []).map((it: any) => ({
          codigo: it.codigo || "",
          descripcion: it.descripcion || "",
          unidad: it.unidad || "NIU",
          cantidad: +it.cantidad || 1,
          valor_unitario: +it.valor_unitario || 0,
          afectacion_igv: it.afectacion_igv || "10",
        }));
        setItems(its);
      })
      .catch(() => {
        toast.error("Comprobante de referencia no encontrado");
        nav("/comprobantes", { replace: true });
      });
  }, [refId, nav]);

  const calc = useMemo(() => calcular(items), [items]);

  function updateItem(i: number, patch: Partial<Item>) {
    setItems((cur) => cur.map((it, idx) => (idx === i ? { ...it, ...patch } : it)));
  }
  function rmItem(i: number) {
    setItems((cur) => cur.filter((_, idx) => idx !== i));
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!ref) return;
    if (!descripcion.trim()) {
      toast.error("El motivo (descripción) es obligatorio");
      return;
    }
    if (items.length === 0) {
      toast.error("La nota debe tener al menos un ítem");
      return;
    }
    setSubmitting(true);
    const path = esCredito ? "/api/v1/notas-credito" : "/api/v1/notas-debito";
    try {
      const r = await api<{ id: string; serie: string; correlativo: number }>(path, {
        body: {
          tipo: tipoCodigo,
          serie,
          fecha_emision: new Date().toISOString().slice(0, 10),
          moneda: ref.moneda,
          receptor: {
            tipo_doc: ref.receptor_tipo_doc,
            num_doc: ref.receptor_doc,
            razon_social: ref.receptor_razon,
          },
          items: items.map((it) => ({
            descripcion: it.descripcion,
            unidad: it.unidad,
            cantidad: +it.cantidad,
            valor_unitario: +it.valor_unitario,
            afectacion_igv: it.afectacion_igv,
          })),
          referencia: {
            tipo_doc: ref.tipo,
            serie: ref.serie,
            correlativo: ref.correlativo,
          },
          motivo_codigo: motivo,
          motivo_descripcion: descripcion,
        },
      });
      toast.success(`${esCredito ? "Nota de crédito" : "Nota de débito"} ${r.serie}-${r.correlativo} encolada`);
      nav(`/comprobantes/${r.id}`);
    } catch (err) {
      const msg = err instanceof ApiError
        ? err.body?.detalle || err.body?.error || err.message
        : String(err);
      toast.error("No se pudo emitir: " + msg);
    } finally {
      setSubmitting(false);
    }
  }

  if (!ref) return <div className="text-slate-500 text-sm">Cargando…</div>;

  return (
    <form onSubmit={onSubmit} className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold text-slate-900">
          Nueva {esCredito ? "nota de crédito" : "nota de débito"}
        </h1>
        <p className="text-sm text-slate-500 mt-1">
          Sobre {ref.tipo === "01" ? "factura" : "boleta"}{" "}
          <strong>{ref.serie}-{ref.correlativo}</strong> · {ref.receptor_razon}
        </p>
      </header>

      <section className="card p-6 space-y-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Motivo</h2>
        <div>
          <label className="label">Tipo de motivo</label>
          <select className="input" value={motivo} onChange={(e) => setMotivo(e.target.value)}>
            {motivos.map((m) => (
              <option key={m.value} value={m.value}>{m.label}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="label">Descripción detallada</label>
          <input
            className="input"
            value={descripcion}
            onChange={(e) => setDescripcion(e.target.value)}
            placeholder="Ej: cliente devolvió 2 unidades por defecto de fabricación"
            required
          />
        </div>
      </section>

      <section className="card p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
            Ítems de la nota
          </h2>
          <div className="text-xs text-slate-500">
            {esCredito ? "El monto de la nota REDUCE el comprobante original." : "El monto de la nota AUMENTA el comprobante original."}
          </div>
        </div>
        <div>
          <label className="label">Serie</label>
          <input
            className="input max-w-xs"
            maxLength={4}
            value={serie}
            onChange={(e) => setSerie(e.target.value.toUpperCase().slice(0, 4))}
          />
        </div>

        {items.map((it, i) => (
          <div key={i} className="rounded-lg border border-slate-200 p-4 bg-slate-50/50">
            <div className="grid grid-cols-12 gap-3">
              <div className="col-span-12 sm:col-span-5">
                <label className="label">Descripción</label>
                <input className="input" value={it.descripcion}
                  onChange={(e) => updateItem(i, { descripcion: e.target.value })} required />
              </div>
              <div className="col-span-4 sm:col-span-2">
                <label className="label">Cantidad</label>
                <input type="number" step="0.01" min="0" className="input" value={it.cantidad}
                  onChange={(e) => updateItem(i, { cantidad: +e.target.value })} required />
              </div>
              <div className="col-span-4 sm:col-span-2">
                <label className="label">V. Unit.</label>
                <input type="number" step="0.01" min="0" className="input" value={it.valor_unitario}
                  onChange={(e) => updateItem(i, { valor_unitario: +e.target.value })} required />
              </div>
              <div className="col-span-4 sm:col-span-2">
                <label className="label">IGV</label>
                <select className="input" value={it.afectacion_igv}
                  onChange={(e) => updateItem(i, { afectacion_igv: e.target.value })}>
                  <option value="10">Gravado 18%</option>
                  <option value="20">Exonerado</option>
                  <option value="30">Inafecto</option>
                </select>
              </div>
              <div className="col-span-12 sm:col-span-1 flex sm:items-end">
                <button type="button" onClick={() => rmItem(i)}
                  className="btn-ghost w-full text-rose-600">✕</button>
              </div>
            </div>
          </div>
        ))}
      </section>

      <section className="card p-6">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 mb-4">Totales</h2>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-sm">
          <div><div className="text-xs text-slate-500">Gravado</div><div className="tabular-nums">{ref.moneda} {calc.gravado.toFixed(2)}</div></div>
          <div><div className="text-xs text-slate-500">Exonerado</div><div className="tabular-nums">{ref.moneda} {calc.exonerado.toFixed(2)}</div></div>
          <div><div className="text-xs text-slate-500">Inafecto</div><div className="tabular-nums">{ref.moneda} {calc.inafecto.toFixed(2)}</div></div>
          <div><div className="text-xs text-slate-500">IGV</div><div className="tabular-nums">{ref.moneda} {calc.igv.toFixed(2)}</div></div>
        </div>
        <div className="mt-4 pt-4 border-t border-slate-200 flex items-baseline justify-between">
          <span className="text-sm text-slate-600">Total de la nota</span>
          <span className="text-2xl font-semibold tabular-nums">{ref.moneda} {calc.total.toFixed(2)}</span>
        </div>
      </section>

      <div className="flex justify-end gap-3">
        <button type="button" className="btn-ghost" onClick={() => nav(-1)}>Cancelar</button>
        <button type="submit" className="btn-primary" disabled={submitting}>
          {submitting ? "Enviando…" : `Emitir ${esCredito ? "nota de crédito" : "nota de débito"}`}
        </button>
      </div>
    </form>
  );
}
