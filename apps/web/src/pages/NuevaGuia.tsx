import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { api, ApiError } from "../api/client";

const MOTIVOS = [
  { value: "01", label: "01 - Venta" },
  { value: "02", label: "02 - Compra" },
  { value: "04", label: "04 - Traslado entre establecimientos" },
  { value: "08", label: "08 - Importación" },
  { value: "09", label: "09 - Exportación" },
  { value: "13", label: "13 - Otros" },
];

type ItemTraslado = {
  codigo?: string;
  descripcion: string;
  unidad: string;
  cantidad: number;
};

const HOY = new Date().toISOString().slice(0, 10);

function emptyItem(): ItemTraslado {
  return { descripcion: "", unidad: "NIU", cantidad: 1 };
}

export function NuevaGuia() {
  const nav = useNavigate();
  const [serie, setSerie] = useState("T001");
  const [fechaEmision, setFechaEmision] = useState(HOY);
  const [fechaTraslado, setFechaTraslado] = useState(HOY);
  // Destinatario
  const [destTipoDoc, setDestTipoDoc] = useState("6");
  const [destNumDoc, setDestNumDoc] = useState("");
  const [destRazon, setDestRazon] = useState("");
  // Origen / destino
  const [origenUbigeo, setOrigenUbigeo] = useState("");
  const [origenDir, setOrigenDir] = useState("");
  const [destinoUbigeo, setDestinoUbigeo] = useState("");
  const [destinoDir, setDestinoDir] = useState("");
  // Traslado
  const [motivo, setMotivo] = useState("01");
  const [modalidad, setModalidad] = useState<"01" | "02">("02");
  const [pesoKg, setPesoKg] = useState(1);
  const [bultos, setBultos] = useState(1);
  // Transportista
  const [transRUC, setTransRUC] = useState("");
  const [transRazon, setTransRazon] = useState("");
  const [placa, setPlaca] = useState("");
  const [docChofer, setDocChofer] = useState("");
  const [nombreChofer, setNombreChofer] = useState("");
  // Items
  const [items, setItems] = useState<ItemTraslado[]>([emptyItem()]);
  const [observaciones, setObservaciones] = useState("");
  const [submitting, setSubmitting] = useState(false);

  function updateItem(i: number, patch: Partial<ItemTraslado>) {
    setItems((cur) => cur.map((it, idx) => (idx === i ? { ...it, ...patch } : it)));
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    const transportista =
      modalidad === "01"
        ? { ruc: transRUC, razon_social: transRazon }
        : {
            placa_vehiculo: placa.toUpperCase().replace(/\s/g, ""),
            doc_chofer: docChofer,
            nombre_chofer: nombreChofer,
          };
    try {
      const r = await api<{ id: string; serie: string; correlativo: number }>("/api/v1/guias", {
        body: {
          serie,
          fecha_emision: fechaEmision,
          fecha_traslado: fechaTraslado,
          destinatario: {
            tipo_doc: destTipoDoc,
            num_doc: destNumDoc,
            razon_social: destRazon,
          },
          punto_partida: { ubigeo: origenUbigeo, direccion: origenDir },
          punto_llegada: { ubigeo: destinoUbigeo, direccion: destinoDir },
          motivo,
          modalidad,
          peso_total_kg: pesoKg,
          num_bultos: bultos,
          transportista,
          items: items.map((it) => ({ ...it, cantidad: +it.cantidad })),
          observaciones,
        },
      });
      toast.success(`Guía ${r.serie}-${r.correlativo} encolada`);
      nav(`/comprobantes/${r.id}`);
    } catch (err) {
      const msg = err instanceof ApiError ? err.body?.detalle || err.body?.error || err.message : String(err);
      toast.error("No se pudo emitir: " + msg);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold text-slate-900">Nueva guía de remisión</h1>
        <p className="text-sm text-slate-500 mt-1">
          Documento que acompaña el traslado físico de mercadería. SUNAT la exige
          obligatoriamente desde julio 2026.
        </p>
      </header>

      <section className="card p-6 space-y-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Datos generales</h2>
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="label">Serie</label>
            <input className="input" value={serie} maxLength={4}
              onChange={(e) => setSerie(e.target.value.toUpperCase().slice(0, 4))} required />
            <p className="text-xs text-slate-400 mt-1">T###</p>
          </div>
          <div>
            <label className="label">Fecha de emisión</label>
            <input type="date" className="input" value={fechaEmision}
              onChange={(e) => setFechaEmision(e.target.value)} required />
          </div>
          <div>
            <label className="label">Fecha de traslado</label>
            <input type="date" className="input" value={fechaTraslado}
              onChange={(e) => setFechaTraslado(e.target.value)} required />
          </div>
        </div>
      </section>

      <section className="card p-6 space-y-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Destinatario</h2>
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="label">Tipo doc.</label>
            <select className="input" value={destTipoDoc} onChange={(e) => setDestTipoDoc(e.target.value)}>
              <option value="6">RUC</option>
              <option value="1">DNI</option>
              <option value="4">Carnet ext.</option>
              <option value="7">Pasaporte</option>
            </select>
          </div>
          <div>
            <label className="label">Número</label>
            <input className="input" value={destNumDoc}
              onChange={(e) => setDestNumDoc(e.target.value.replace(/\D/g, ""))} required />
          </div>
          <div>
            <label className="label">Razón / Nombre</label>
            <input className="input" value={destRazon}
              onChange={(e) => setDestRazon(e.target.value)} required />
          </div>
        </div>
      </section>

      <section className="card p-6 space-y-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Origen y destino</h2>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div className="space-y-3">
            <div className="font-medium text-slate-700 text-sm">Punto de partida</div>
            <div>
              <label className="label">Ubigeo (6 dígitos)</label>
              <input className="input" value={origenUbigeo} maxLength={6}
                onChange={(e) => setOrigenUbigeo(e.target.value.replace(/\D/g, ""))} required />
            </div>
            <div>
              <label className="label">Dirección</label>
              <input className="input" value={origenDir}
                onChange={(e) => setOrigenDir(e.target.value)} required />
            </div>
          </div>
          <div className="space-y-3">
            <div className="font-medium text-slate-700 text-sm">Punto de llegada</div>
            <div>
              <label className="label">Ubigeo (6 dígitos)</label>
              <input className="input" value={destinoUbigeo} maxLength={6}
                onChange={(e) => setDestinoUbigeo(e.target.value.replace(/\D/g, ""))} required />
            </div>
            <div>
              <label className="label">Dirección</label>
              <input className="input" value={destinoDir}
                onChange={(e) => setDestinoDir(e.target.value)} required />
            </div>
          </div>
        </div>
      </section>

      <section className="card p-6 space-y-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Traslado</h2>
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="label">Motivo</label>
            <select className="input" value={motivo} onChange={(e) => setMotivo(e.target.value)}>
              {MOTIVOS.map((m) => <option key={m.value} value={m.value}>{m.label}</option>)}
            </select>
          </div>
          <div>
            <label className="label">Modalidad</label>
            <select className="input" value={modalidad}
              onChange={(e) => setModalidad(e.target.value as "01" | "02")}>
              <option value="02">02 - Privado (lo hace mi empresa)</option>
              <option value="01">01 - Público (lo hace un transportista)</option>
            </select>
          </div>
          <div>
            <label className="label">Peso total (kg)</label>
            <input type="number" step="0.01" min="0" className="input" value={pesoKg}
              onChange={(e) => setPesoKg(+e.target.value)} required />
          </div>
        </div>
        <div>
          <label className="label">Número de bultos</label>
          <input type="number" min="0" className="input max-w-xs" value={bultos}
            onChange={(e) => setBultos(+e.target.value)} />
        </div>

        {modalidad === "01" ? (
          <div className="rounded-lg border border-slate-200 p-4 bg-slate-50/50 space-y-3">
            <div className="font-medium text-slate-700 text-sm">Transportista (público)</div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label">RUC del transportista</label>
                <input className="input" value={transRUC} maxLength={11}
                  onChange={(e) => setTransRUC(e.target.value.replace(/\D/g, ""))} required />
              </div>
              <div>
                <label className="label">Razón social</label>
                <input className="input" value={transRazon}
                  onChange={(e) => setTransRazon(e.target.value)} required />
              </div>
            </div>
          </div>
        ) : (
          <div className="rounded-lg border border-slate-200 p-4 bg-slate-50/50 space-y-3">
            <div className="font-medium text-slate-700 text-sm">Vehículo y chofer (transporte privado)</div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="label">Placa del vehículo</label>
                <input className="input" value={placa}
                  onChange={(e) => setPlaca(e.target.value.toUpperCase())}
                  placeholder="ABC-123" required />
              </div>
              <div>
                <label className="label">DNI/CE del chofer</label>
                <input className="input" value={docChofer}
                  onChange={(e) => setDocChofer(e.target.value)} required />
              </div>
              <div>
                <label className="label">Nombre del chofer</label>
                <input className="input" value={nombreChofer}
                  onChange={(e) => setNombreChofer(e.target.value)} required />
              </div>
            </div>
          </div>
        )}
      </section>

      <section className="card p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Mercadería</h2>
          <button type="button" className="btn-ghost text-xs"
            onClick={() => setItems((cur) => [...cur, emptyItem()])}>
            + Agregar ítem
          </button>
        </div>
        {items.map((it, i) => (
          <div key={i} className="rounded-lg border border-slate-200 p-4 bg-slate-50/50">
            <div className="grid grid-cols-12 gap-3">
              <div className="col-span-12 sm:col-span-6">
                <label className="label">Descripción</label>
                <input className="input" value={it.descripcion}
                  onChange={(e) => updateItem(i, { descripcion: e.target.value })} required />
              </div>
              <div className="col-span-4 sm:col-span-2">
                <label className="label">Código</label>
                <input className="input" value={it.codigo || ""}
                  onChange={(e) => updateItem(i, { codigo: e.target.value })} />
              </div>
              <div className="col-span-4 sm:col-span-2">
                <label className="label">Unidad</label>
                <select className="input" value={it.unidad}
                  onChange={(e) => updateItem(i, { unidad: e.target.value })}>
                  <option value="NIU">NIU</option>
                  <option value="KGM">KGM</option>
                  <option value="LTR">LTR</option>
                  <option value="BX">BX</option>
                  <option value="ZZ">ZZ</option>
                </select>
              </div>
              <div className="col-span-3 sm:col-span-1">
                <label className="label">Cantidad</label>
                <input type="number" step="0.01" min="0" className="input" value={it.cantidad}
                  onChange={(e) => updateItem(i, { cantidad: +e.target.value })} required />
              </div>
              <div className="col-span-1 flex sm:items-end">
                <button type="button"
                  onClick={() => setItems((cur) => cur.length === 1 ? cur : cur.filter((_, idx) => idx !== i))}
                  className="btn-ghost w-full text-rose-600"
                  disabled={items.length === 1}>✕</button>
              </div>
            </div>
          </div>
        ))}
      </section>

      <section className="card p-6">
        <label className="label">Observaciones (opcional)</label>
        <textarea
          className="input"
          rows={2}
          value={observaciones}
          onChange={(e) => setObservaciones(e.target.value)}
        />
      </section>

      <div className="flex justify-end gap-3">
        <button type="button" className="btn-ghost" onClick={() => nav(-1)}>Cancelar</button>
        <button type="submit" className="btn-primary" disabled={submitting}>
          {submitting ? "Enviando…" : "Emitir guía"}
        </button>
      </div>
    </form>
  );
}
