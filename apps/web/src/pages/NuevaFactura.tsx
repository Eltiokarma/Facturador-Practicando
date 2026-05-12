import { FormEvent, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { api, ApiError } from "../api/client";
import { Autocomplete } from "../components/Autocomplete";
import { Cliente, Item, Producto } from "../types";

type Tipo = "01" | "03";

const HOY = new Date().toISOString().slice(0, 10);

function emptyItem(): Item {
  return {
    codigo: "",
    descripcion: "",
    unidad: "NIU",
    cantidad: 1,
    valor_unitario: 0,
    afectacion_igv: "10",
  };
}

function calcular(items: Item[]) {
  let gravado = 0, exonerado = 0, inafecto = 0, igv = 0;
  const lineas = items.map((it) => {
    const sub = round2((+it.cantidad || 0) * (+it.valor_unitario || 0));
    let lineIgv = 0;
    let precioUnit = +it.valor_unitario || 0;
    switch (it.afectacion_igv) {
      case "10":
        lineIgv = round2(sub * 0.18);
        precioUnit = round2(precioUnit * 1.18);
        gravado += sub;
        igv += lineIgv;
        break;
      case "20":
        exonerado += sub;
        break;
      case "30":
        inafecto += sub;
        break;
    }
    return { sub, igv: lineIgv, precioUnit, total: round2(sub + lineIgv) };
  });
  const total = round2(gravado + exonerado + inafecto + igv);
  return {
    lineas,
    totales: {
      gravado: round2(gravado),
      exonerado: round2(exonerado),
      inafecto: round2(inafecto),
      igv: round2(igv),
      total,
    },
  };
}

function round2(v: number): number {
  return Math.round(v * 100) / 100;
}

export function NuevaFactura() {
  const nav = useNavigate();
  const [tipo, setTipo] = useState<Tipo>("01");
  const [serie, setSerie] = useState("F001");
  const [fecha, setFecha] = useState(HOY);
  const [moneda, setMoneda] = useState("PEN");
  const [receptorTipoDoc, setReceptorTipoDoc] = useState("6");
  const [receptorNumDoc, setReceptorNumDoc] = useState("");
  const [receptorRazon, setReceptorRazon] = useState("");
  const [receptorDir, setReceptorDir] = useState("");
  const [items, setItems] = useState<Item[]>([emptyItem()]);
  const [submitting, setSubmitting] = useState(false);
  const [guardarCliente, setGuardarCliente] = useState(true);

  const calc = useMemo(() => calcular(items), [items]);

  function setTipoCambiar(t: Tipo) {
    setTipo(t);
    if (t === "01") {
      setReceptorTipoDoc("6");
      if (!serie.startsWith("F")) setSerie("F001");
    } else {
      setReceptorTipoDoc("1");
      if (!serie.startsWith("B")) setSerie("B001");
    }
  }

  function updateItem(idx: number, patch: Partial<Item>) {
    setItems((cur) => cur.map((it, i) => (i === idx ? { ...it, ...patch } : it)));
  }
  function addItem() { setItems((cur) => [...cur, emptyItem()]); }
  function rmItem(idx: number) {
    setItems((cur) => cur.length === 1 ? cur : cur.filter((_, i) => i !== idx));
  }

  async function buscarClientes(q: string): Promise<Cliente[]> {
    const params = new URLSearchParams({ limit: "8" });
    if (q) params.set("q", q);
    const r = await api<{ items: Cliente[] }>(`/api/v1/clientes?${params}`);
    return r.items || [];
  }
  function pickCliente(c: Cliente) {
    setReceptorTipoDoc(c.tipo_doc);
    setReceptorNumDoc(c.num_doc);
    setReceptorRazon(c.razon_social);
    setReceptorDir(c.direccion || "");
  }

  async function buscarProductos(q: string): Promise<Producto[]> {
    const params = new URLSearchParams({ limit: "8" });
    if (q) params.set("q", q);
    const r = await api<{ items: Producto[] }>(`/api/v1/productos?${params}`);
    return r.items || [];
  }
  function pickProducto(idx: number, p: Producto) {
    updateItem(idx, {
      codigo: p.codigo || "",
      descripcion: p.descripcion,
      unidad: p.unidad,
      valor_unitario: p.valor_unitario,
      afectacion_igv: p.afectacion_igv,
    });
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (items.length === 0) {
      toast.error("Agregá al menos un ítem");
      return;
    }
    for (const it of items) {
      if (!it.descripcion.trim()) {
        toast.error("Hay un ítem sin descripción");
        return;
      }
      if (it.cantidad <= 0) {
        toast.error("Hay un ítem con cantidad 0");
        return;
      }
    }
    setSubmitting(true);
    const path = tipo === "01" ? "/api/v1/facturas" : "/api/v1/boletas";
    try {
      const resp = await api<{
        id: string; tipo: string; serie: string; correlativo: number;
        estado: string;
      }>(path, {
        body: {
          tipo,
          serie,
          fecha_emision: fecha,
          moneda,
          receptor: {
            tipo_doc: receptorTipoDoc,
            num_doc: receptorNumDoc.trim(),
            razon_social: receptorRazon.trim(),
            direccion: receptorDir.trim(),
          },
          items: items.map((it) => ({
            codigo: it.codigo || undefined,
            descripcion: it.descripcion,
            unidad: it.unidad,
            cantidad: +it.cantidad,
            valor_unitario: +it.valor_unitario,
            afectacion_igv: it.afectacion_igv,
          })),
        },
      });

      // Guardar al catálogo en background si está la opción (no esperamos la respuesta de SUNAT)
      if (guardarCliente && receptorNumDoc.trim() && receptorRazon.trim()) {
        api("/api/v1/clientes", {
          body: {
            tipo_doc: receptorTipoDoc,
            num_doc: receptorNumDoc.trim(),
            razon_social: receptorRazon.trim(),
            direccion: receptorDir.trim() || undefined,
          },
        }).catch(() => { /* best effort */ });
      }

      toast.success(`Encolado: ${resp.serie}-${resp.correlativo}. Esperando respuesta de SUNAT…`);
      nav(`/comprobantes/${resp.id}`);
    } catch (err) {
      const msg = err instanceof ApiError
        ? err.body?.detalle || err.body?.error || err.message
        : String(err);
      toast.error("No se pudo emitir: " + msg);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold text-slate-900">Nuevo comprobante</h1>
        <p className="text-sm text-slate-500 mt-1">
          El sistema recalcula totales server-side — esto es previsualización.
        </p>
      </header>

      <section className="card p-6 space-y-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Tipo y serie</h2>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <div>
            <label className="label">Tipo</label>
            <select className="input" value={tipo} onChange={(e) => setTipoCambiar(e.target.value as Tipo)}>
              <option value="01">Factura</option>
              <option value="03">Boleta</option>
            </select>
          </div>
          <div>
            <label className="label">Serie</label>
            <input className="input" value={serie} maxLength={4} required
              onChange={(e) => setSerie(e.target.value.toUpperCase().slice(0, 4))} />
          </div>
          <div>
            <label className="label">Fecha emisión</label>
            <input type="date" className="input" value={fecha}
              onChange={(e) => setFecha(e.target.value)} required />
          </div>
          <div>
            <label className="label">Moneda</label>
            <select className="input" value={moneda} onChange={(e) => setMoneda(e.target.value)}>
              <option value="PEN">PEN (Soles)</option>
              <option value="USD">USD (Dólares)</option>
            </select>
          </div>
        </div>
      </section>

      <section className="card p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Cliente</h2>
          <label className="text-xs text-slate-500 flex items-center gap-2">
            <input
              type="checkbox"
              checked={guardarCliente}
              onChange={(e) => setGuardarCliente(e.target.checked)}
              className="rounded"
            />
            Guardar al catálogo
          </label>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label className="label">Tipo doc.</label>
            <select className="input" value={receptorTipoDoc}
              onChange={(e) => setReceptorTipoDoc(e.target.value)}
              disabled={tipo === "01"}>
              {tipo === "01" ? (
                <option value="6">RUC</option>
              ) : (
                <>
                  <option value="1">DNI</option>
                  <option value="0">Sin documento</option>
                  <option value="4">Carnet ext.</option>
                  <option value="7">Pasaporte</option>
                  <option value="6">RUC</option>
                </>
              )}
            </select>
          </div>
          <div>
            <label className="label">Número</label>
            <Autocomplete<Cliente>
              placeholder={tipo === "01" ? "20XXXXXXXXX" : "12345678"}
              value={receptorNumDoc}
              onValueChange={(v) => setReceptorNumDoc(v.replace(/\D/g, ""))}
              search={(q) => buscarClientes(q)}
              keyOf={(c) => c.id || c.num_doc}
              renderOption={(c) => (
                <>
                  <div className="font-medium text-slate-900">{c.razon_social}</div>
                  <div className="text-xs text-slate-500">{c.num_doc}</div>
                </>
              )}
              onSelect={pickCliente}
            />
          </div>
          <div>
            <label className="label">Razón / Nombre</label>
            <input className="input" value={receptorRazon}
              onChange={(e) => setReceptorRazon(e.target.value)} required />
          </div>
        </div>
        <div>
          <label className="label">Dirección (opcional)</label>
          <input className="input" value={receptorDir}
            onChange={(e) => setReceptorDir(e.target.value)} />
        </div>
      </section>

      <section className="card p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Ítems</h2>
          <button type="button" className="btn-ghost text-xs" onClick={addItem}>+ Agregar ítem</button>
        </div>
        <div className="space-y-3">
          {items.map((it, idx) => (
            <div key={idx} className="rounded-lg border border-slate-200 p-4 bg-slate-50/50">
              <div className="grid grid-cols-12 gap-3">
                <div className="col-span-12 sm:col-span-5">
                  <label className="label">Descripción</label>
                  <Autocomplete<Producto>
                    placeholder="Buscar producto o escribir uno nuevo…"
                    value={it.descripcion}
                    onValueChange={(v) => updateItem(idx, { descripcion: v })}
                    search={(q) => buscarProductos(q)}
                    keyOf={(p) => p.id || p.descripcion}
                    renderOption={(p) => (
                      <div className="flex items-center justify-between gap-2">
                        <div>
                          <div className="font-medium text-slate-900">{p.descripcion}</div>
                          {p.codigo && <div className="text-xs text-slate-500">{p.codigo}</div>}
                        </div>
                        <div className="text-sm text-slate-700 tabular-nums">
                          {p.valor_unitario.toFixed(2)}
                        </div>
                      </div>
                    )}
                    onSelect={(p) => pickProducto(idx, p)}
                  />
                </div>
                <div className="col-span-4 sm:col-span-2">
                  <label className="label">Cantidad</label>
                  <input type="number" step="0.01" min="0" className="input"
                    value={it.cantidad}
                    onChange={(e) => updateItem(idx, { cantidad: +e.target.value })} required />
                </div>
                <div className="col-span-4 sm:col-span-2">
                  <label className="label">V. Unit.</label>
                  <input type="number" step="0.01" min="0" className="input"
                    value={it.valor_unitario}
                    onChange={(e) => updateItem(idx, { valor_unitario: +e.target.value })} required />
                </div>
                <div className="col-span-4 sm:col-span-2">
                  <label className="label">IGV</label>
                  <select className="input" value={it.afectacion_igv}
                    onChange={(e) => updateItem(idx, { afectacion_igv: e.target.value })}>
                    <option value="10">Gravado 18%</option>
                    <option value="20">Exonerado</option>
                    <option value="30">Inafecto</option>
                  </select>
                </div>
                <div className="col-span-12 sm:col-span-1 flex sm:items-end">
                  <button type="button" onClick={() => rmItem(idx)}
                    className="btn-ghost w-full text-rose-600"
                    disabled={items.length === 1} title="Quitar ítem">✕</button>
                </div>
                <div className="col-span-12 text-right text-sm text-slate-600 tabular-nums">
                  Subtotal {calc.lineas[idx]?.sub.toFixed(2)} · IGV{" "}
                  {calc.lineas[idx]?.igv.toFixed(2)} · Total{" "}
                  <strong>{moneda} {calc.lineas[idx]?.total.toFixed(2)}</strong>
                </div>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="card p-6">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 mb-4">
          Totales (previsualización)
        </h2>
        <dl className="grid grid-cols-2 sm:grid-cols-4 gap-y-3 gap-x-6 text-sm">
          <Row label="Gravado" value={`${moneda} ${calc.totales.gravado.toFixed(2)}`} />
          <Row label="Exonerado" value={`${moneda} ${calc.totales.exonerado.toFixed(2)}`} />
          <Row label="Inafecto" value={`${moneda} ${calc.totales.inafecto.toFixed(2)}`} />
          <Row label="IGV (18%)" value={`${moneda} ${calc.totales.igv.toFixed(2)}`} />
        </dl>
        <div className="mt-4 pt-4 border-t border-slate-200 flex items-baseline justify-between">
          <dt className="text-sm font-medium text-slate-600">Total a cobrar</dt>
          <dd className="text-2xl font-semibold text-slate-900 tabular-nums">
            {moneda} {calc.totales.total.toFixed(2)}
          </dd>
        </div>
      </section>

      <div className="flex items-center justify-end gap-3">
        <button type="button" className="btn-ghost" onClick={() => nav(-1)}>Cancelar</button>
        <button type="submit" className="btn-primary" disabled={submitting}>
          {submitting ? "Enviando a SUNAT…" : "Emitir comprobante"}
        </button>
      </div>
    </form>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col">
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd className="text-sm text-slate-900 tabular-nums">{value}</dd>
    </div>
  );
}
