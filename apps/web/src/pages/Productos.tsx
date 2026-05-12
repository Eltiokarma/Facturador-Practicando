import { FormEvent, useEffect, useState } from "react";
import { toast } from "sonner";
import { api } from "../api/client";
import { Producto } from "../types";
import { Modal } from "./Clientes";

const UNIDADES = [
  { value: "NIU", label: "NIU — Unidad (bienes)" },
  { value: "ZZ", label: "ZZ — Servicio" },
  { value: "KGM", label: "KGM — Kilogramo" },
  { value: "MTR", label: "MTR — Metro" },
  { value: "LTR", label: "LTR — Litro" },
  { value: "BX", label: "BX — Caja" },
  { value: "DZN", label: "DZN — Docena" },
  { value: "HUR", label: "HUR — Hora" },
];

const AFECTACIONES = [
  { value: "10", label: "Gravado (18% IGV)" },
  { value: "20", label: "Exonerado" },
  { value: "30", label: "Inafecto" },
];

export function Productos() {
  const [items, setItems] = useState<Producto[] | null>(null);
  const [q, setQ] = useState("");
  const [editing, setEditing] = useState<Producto | null>(null);

  async function reload() {
    const params = new URLSearchParams({ limit: "200" });
    if (q) params.set("q", q);
    const r = await api<{ items: Producto[] }>(`/api/v1/productos?${params}`);
    setItems(r.items || []);
  }

  useEffect(() => {
    reload().catch(() => toast.error("No pude cargar productos"));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q]);

  async function onDelete(p: Producto) {
    if (!p.id) return;
    if (!confirm(`¿Eliminar "${p.descripcion}"?`)) return;
    try {
      await api(`/api/v1/productos/${p.id}`, { method: "DELETE" });
      toast.success("Producto eliminado");
      reload();
    } catch (e: any) {
      toast.error(e.message);
    }
  }

  return (
    <div className="space-y-6">
      <header className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">Productos</h1>
          <p className="text-sm text-slate-500 mt-1">
            Tu catálogo: lo que vendés con precio y unidad. Se autocompletan en la emisión.
          </p>
        </div>
        <button
          className="btn-primary"
          onClick={() => setEditing({
            descripcion: "", unidad: "NIU",
            valor_unitario: 0, afectacion_igv: "10",
          })}
        >
          + Nuevo producto
        </button>
      </header>

      <div className="card p-4">
        <input
          className="input"
          placeholder="Buscar por descripción o código…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      <div className="card overflow-hidden">
        {!items && <div className="p-10 text-center text-slate-500 text-sm">Cargando…</div>}
        {items && items.length === 0 && (
          <div className="p-10 text-center text-slate-500 text-sm">
            {q ? "Sin resultados." : "No hay productos en el catálogo."}
          </div>
        )}
        {items && items.length > 0 && (
          <table className="min-w-full divide-y divide-slate-100 text-sm">
            <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-4 py-3 text-left">Producto / servicio</th>
                <th className="px-4 py-3 text-left">Unidad</th>
                <th className="px-4 py-3 text-right">V. Unit. (sin IGV)</th>
                <th className="px-4 py-3 text-left">IGV</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {items.map((p) => (
                <tr key={p.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3">
                    <div className="font-medium text-slate-900">{p.descripcion}</div>
                    {p.codigo && <div className="text-xs text-slate-500">Código: {p.codigo}</div>}
                  </td>
                  <td className="px-4 py-3 text-slate-700">{p.unidad}</td>
                  <td className="px-4 py-3 text-right tabular-nums">
                    {p.valor_unitario.toFixed(2)}
                  </td>
                  <td className="px-4 py-3">
                    {AFECTACIONES.find((a) => a.value === p.afectacion_igv)?.label || p.afectacion_igv}
                  </td>
                  <td className="px-4 py-3 text-right space-x-3">
                    <button className="text-brand-600 hover:underline text-sm" onClick={() => setEditing(p)}>
                      Editar
                    </button>
                    <button className="text-rose-600 hover:underline text-sm" onClick={() => onDelete(p)}>
                      Eliminar
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {editing && (
        <ProductoForm
          initial={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            reload();
          }}
        />
      )}
    </div>
  );
}

function ProductoForm({
  initial, onClose, onSaved,
}: { initial: Producto; onClose: () => void; onSaved: () => void }) {
  const [p, setP] = useState<Producto>(initial);
  const [saving, setSaving] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!p.descripcion.trim()) {
      toast.error("Descripción es obligatoria");
      return;
    }
    setSaving(true);
    try {
      const path = p.id ? `/api/v1/productos/${p.id}` : `/api/v1/productos`;
      await api(path, { method: p.id ? "PUT" : "POST", body: p });
      toast.success(p.id ? "Producto actualizado" : "Producto creado");
      onSaved();
    } catch (e: any) {
      toast.error(e?.body?.detalle || "No se pudo guardar");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal title={p.id ? "Editar producto" : "Nuevo producto"} onClose={onClose}>
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label className="label">Descripción</label>
          <input
            className="input"
            value={p.descripcion}
            onChange={(e) => setP({ ...p, descripcion: e.target.value })}
            required
            autoFocus
          />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">Código (SKU)</label>
            <input
              className="input"
              value={p.codigo || ""}
              onChange={(e) => setP({ ...p, codigo: e.target.value })}
            />
          </div>
          <div>
            <label className="label">Unidad</label>
            <select
              className="input"
              value={p.unidad}
              onChange={(e) => setP({ ...p, unidad: e.target.value })}
            >
              {UNIDADES.map((u) => (
                <option key={u.value} value={u.value}>{u.label}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">Valor unitario (sin IGV)</label>
            <input
              className="input"
              type="number"
              step="0.01"
              min="0"
              value={p.valor_unitario}
              onChange={(e) => setP({ ...p, valor_unitario: +e.target.value })}
              required
            />
          </div>
          <div>
            <label className="label">Afectación IGV</label>
            <select
              className="input"
              value={p.afectacion_igv}
              onChange={(e) => setP({ ...p, afectacion_igv: e.target.value })}
            >
              {AFECTACIONES.map((a) => (
                <option key={a.value} value={a.value}>{a.label}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="flex justify-end gap-2 pt-2">
          <button type="button" className="btn-ghost" onClick={onClose}>Cancelar</button>
          <button type="submit" className="btn-primary" disabled={saving}>
            {saving ? "Guardando…" : "Guardar"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
