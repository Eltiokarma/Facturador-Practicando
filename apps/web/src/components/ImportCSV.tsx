import { FormEvent, useState } from "react";
import { toast } from "sonner";
import { Modal } from "../pages/Clientes";

const BASE_API = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

type Resultado = {
  procesados: number;
  creados: number;
  errores?: { fila: number; mensaje: string }[];
};

export function ImportCSV({
  path, columnas, ejemplo, onClose, onDone,
}: {
  path: string;
  columnas: { name: string; required?: boolean; help?: string }[];
  ejemplo: string;
  onClose: () => void;
  onDone: () => void;
}) {
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [resultado, setResultado] = useState<Resultado | null>(null);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!file) {
      toast.error("Seleccioná un archivo CSV");
      return;
    }
    const tokens = JSON.parse(localStorage.getItem("facturador.tokens") || "null");
    if (!tokens?.access) {
      toast.error("Sesión expirada");
      return;
    }
    setUploading(true);
    setResultado(null);
    const fd = new FormData();
    fd.append("file", file);
    try {
      const r = await fetch(`${BASE_API}${path}`, {
        method: "POST",
        headers: { Authorization: `Bearer ${tokens.access}` },
        body: fd,
      });
      const body = await r.json();
      if (!r.ok) throw new Error(body?.detalle || body?.error || `HTTP ${r.status}`);
      setResultado(body);
      if ((body.errores?.length || 0) === 0) {
        toast.success(`Importados ${body.creados} de ${body.procesados}`);
        onDone();
      } else {
        toast.warning(`Importados ${body.creados} de ${body.procesados} (${body.errores.length} con error)`);
        onDone();
      }
    } catch (err: any) {
      toast.error("No se pudo importar: " + (err?.message || "error"));
    } finally {
      setUploading(false);
    }
  }

  function descargarPlantilla() {
    const blob = new Blob([ejemplo], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "plantilla.csv";
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <Modal title="Importar desde CSV" onClose={onClose}>
      <form onSubmit={onSubmit} className="space-y-4">
        <div className="rounded-lg bg-slate-50 border border-slate-200 p-3 text-xs">
          <div className="font-medium text-slate-700 mb-2">Columnas esperadas (primera fila = header):</div>
          <ul className="space-y-1 list-disc list-inside">
            {columnas.map((c) => (
              <li key={c.name}>
                <code className="font-mono">{c.name}</code>
                {c.required ? <span className="text-rose-600"> *</span> : <span className="text-slate-400"> (opcional)</span>}
                {c.help && <span className="text-slate-500"> — {c.help}</span>}
              </li>
            ))}
          </ul>
          <button
            type="button"
            onClick={descargarPlantilla}
            className="mt-3 text-brand-600 hover:underline text-xs"
          >
            ⬇ Descargar plantilla
          </button>
        </div>

        <div>
          <label className="label">Archivo CSV (utf-8, separado por comas)</label>
          <input
            type="file"
            accept=".csv,text/csv"
            onChange={(e) => setFile(e.target.files?.[0] || null)}
            className="block w-full text-sm text-slate-700 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:bg-brand-50 file:text-brand-700 hover:file:bg-brand-100 cursor-pointer"
          />
          {file && (
            <p className="text-xs text-slate-500 mt-1">
              Seleccionado: {file.name} ({Math.round(file.size / 1024)} KB)
            </p>
          )}
        </div>

        {resultado && (
          <div className="rounded-lg p-3 text-sm space-y-2">
            <div>
              Procesadas: <strong>{resultado.procesados}</strong> · Creadas/actualizadas:{" "}
              <strong className="text-emerald-700">{resultado.creados}</strong>
              {resultado.errores && resultado.errores.length > 0 && (
                <span> · Con error: <strong className="text-rose-600">{resultado.errores.length}</strong></span>
              )}
            </div>
            {resultado.errores && resultado.errores.length > 0 && (
              <div className="max-h-32 overflow-auto bg-rose-50 border border-rose-200 rounded p-2 text-xs space-y-1">
                {resultado.errores.map((er, i) => (
                  <div key={i}>
                    <span className="font-mono">fila {er.fila}:</span> {er.mensaje}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        <div className="flex justify-end gap-2 pt-2">
          <button type="button" className="btn-ghost" onClick={onClose}>Cerrar</button>
          <button type="submit" className="btn-primary" disabled={uploading || !file}>
            {uploading ? "Importando…" : "Importar"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
