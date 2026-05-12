import { FormEvent, useEffect, useState } from "react";
import { toast } from "sonner";
import { api } from "../api/client";
import { Cliente } from "../types";

const TIPOS_DOC = [
  { value: "6", label: "RUC" },
  { value: "1", label: "DNI" },
  { value: "4", label: "Carnet extranjería" },
  { value: "7", label: "Pasaporte" },
  { value: "0", label: "Sin documento" },
];

function tipoLabel(t: string) {
  return TIPOS_DOC.find((x) => x.value === t)?.label || t;
}

export function Clientes() {
  const [items, setItems] = useState<Cliente[] | null>(null);
  const [q, setQ] = useState("");
  const [editing, setEditing] = useState<Cliente | null>(null);

  async function reload() {
    const params = new URLSearchParams({ limit: "200" });
    if (q) params.set("q", q);
    const r = await api<{ items: Cliente[] }>(`/api/v1/clientes?${params}`);
    setItems(r.items || []);
  }

  useEffect(() => {
    reload().catch(() => toast.error("No pude cargar clientes"));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q]);

  async function onDelete(c: Cliente) {
    if (!c.id) return;
    if (!confirm(`¿Eliminar a ${c.razon_social}?`)) return;
    try {
      await api(`/api/v1/clientes/${c.id}`, { method: "DELETE" });
      toast.success("Cliente eliminado");
      reload();
    } catch (e: any) {
      toast.error(e.message || "No se pudo eliminar");
    }
  }

  return (
    <div className="space-y-6">
      <header className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">Clientes</h1>
          <p className="text-sm text-slate-500 mt-1">
            Acá guardás los datos para no escribirlos cada vez que emitas.
          </p>
        </div>
        <button className="btn-primary" onClick={() => setEditing({ tipo_doc: "6", num_doc: "", razon_social: "" })}>
          + Nuevo cliente
        </button>
      </header>

      <div className="card p-4">
        <input
          className="input"
          placeholder="Buscar por nombre o documento…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      <div className="card overflow-hidden">
        {!items && <div className="p-10 text-center text-slate-500 text-sm">Cargando…</div>}
        {items && items.length === 0 && (
          <div className="p-10 text-center text-slate-500 text-sm">
            {q ? "Sin resultados." : "No hay clientes guardados todavía."}
          </div>
        )}
        {items && items.length > 0 && (
          <table className="min-w-full divide-y divide-slate-100 text-sm">
            <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-4 py-3 text-left">Cliente</th>
                <th className="px-4 py-3 text-left">Documento</th>
                <th className="px-4 py-3 text-left">Contacto</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {items.map((c) => (
                <tr key={c.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3">
                    <div className="font-medium text-slate-900">{c.razon_social}</div>
                    {c.direccion && (
                      <div className="text-xs text-slate-500">{c.direccion}</div>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <div className="text-slate-700">{tipoLabel(c.tipo_doc)}</div>
                    <div className="font-mono text-xs text-slate-500">{c.num_doc}</div>
                  </td>
                  <td className="px-4 py-3 text-slate-600 text-xs">
                    {c.email && <div>{c.email}</div>}
                    {c.telefono && <div>{c.telefono}</div>}
                  </td>
                  <td className="px-4 py-3 text-right space-x-3">
                    <button
                      className="text-brand-600 hover:underline text-sm"
                      onClick={() => setEditing(c)}
                    >
                      Editar
                    </button>
                    <button
                      className="text-rose-600 hover:underline text-sm"
                      onClick={() => onDelete(c)}
                    >
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
        <ClienteForm
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

function ClienteForm({
  initial, onClose, onSaved,
}: { initial: Cliente; onClose: () => void; onSaved: () => void }) {
  const [c, setC] = useState<Cliente>(initial);
  const [saving, setSaving] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!c.num_doc.trim() || !c.razon_social.trim()) {
      toast.error("Documento y nombre son obligatorios");
      return;
    }
    setSaving(true);
    try {
      const path = c.id ? `/api/v1/clientes/${c.id}` : `/api/v1/clientes`;
      await api(path, { method: c.id ? "PUT" : "POST", body: c });
      toast.success(c.id ? "Cliente actualizado" : "Cliente creado");
      onSaved();
    } catch (e: any) {
      toast.error(e?.body?.detalle || "No se pudo guardar");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal title={c.id ? "Editar cliente" : "Nuevo cliente"} onClose={onClose}>
      <form onSubmit={onSubmit} className="space-y-4">
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="label">Tipo doc.</label>
            <select
              className="input"
              value={c.tipo_doc}
              onChange={(e) => setC({ ...c, tipo_doc: e.target.value })}
            >
              {TIPOS_DOC.map((t) => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
          </div>
          <div className="col-span-2">
            <label className="label">Número</label>
            <input
              className="input"
              value={c.num_doc}
              onChange={(e) => setC({ ...c, num_doc: e.target.value.replace(/\D/g, "") })}
              required
            />
          </div>
        </div>
        <div>
          <label className="label">Razón social / Nombre</label>
          <input className="input" value={c.razon_social}
            onChange={(e) => setC({ ...c, razon_social: e.target.value })} required />
        </div>
        <div>
          <label className="label">Dirección</label>
          <input className="input" value={c.direccion || ""}
            onChange={(e) => setC({ ...c, direccion: e.target.value })} />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">Email</label>
            <input className="input" type="email" value={c.email || ""}
              onChange={(e) => setC({ ...c, email: e.target.value })} />
          </div>
          <div>
            <label className="label">Teléfono</label>
            <input className="input" value={c.telefono || ""}
              onChange={(e) => setC({ ...c, telefono: e.target.value })} />
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

export function Modal({
  title, onClose, children,
}: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
      onClick={onClose}
    >
      <div
        className="bg-white rounded-xl shadow-xl w-full max-w-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-6 py-4 border-b border-slate-100 flex items-center justify-between">
          <h2 className="text-lg font-medium text-slate-900">{title}</h2>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-600">✕</button>
        </div>
        <div className="px-6 py-5">{children}</div>
      </div>
    </div>
  );
}
