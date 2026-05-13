import { FormEvent, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";
import { api, apiBlob } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { notify } from "../services/notifications";
import {
  detectCapability,
  getPrinterSettings,
  getSavedPrinter,
  printBytes,
} from "../services/printer";
import { buildTicket } from "../services/ticket-template";
import { ComprobanteDetalleT } from "../types";
import { Modal } from "./Clientes";
import { EstadoBadge, labelTipo } from "./Dashboard";

const ESTADOS_EN_PROCESO = new Set(["pendiente", "enviando", "error"]);

export function ComprobanteDetalle() {
  const { id } = useParams<{ id: string }>();
  const nav = useNavigate();
  const [c, setC] = useState<ComprobanteDetalleT | null>(null);
  const [loading, setLoading] = useState(true);
  const [showPayload, setShowPayload] = useState(false);
  const [showAnular, setShowAnular] = useState(false);
  const [printing, setPrinting] = useState(false);
  const { tenants, user } = useAuth();
  const lastEstado = useRef<string | null>(null);
  const autoPrintedRef = useRef(false); // que no se imprima dos veces

  useEffect(() => {
    if (!id) return;
    let cancelled = false;

    async function fetchOne() {
      try {
        const data = await api<ComprobanteDetalleT>(`/api/v1/comprobantes/${id}`);
        if (cancelled) return;
        setC(data);
        // Notificar cuando llega la respuesta final de SUNAT
        if (lastEstado.current && ESTADOS_EN_PROCESO.has(lastEstado.current)) {
          const numero = `${data.serie}-${data.correlativo}`;
          if (data.estado === "aceptado") {
            toast.success(`SUNAT aceptó ${numero}`);
            notify("Comprobante aceptado", `${numero} fue aceptado por SUNAT`);
          } else if (data.estado === "aceptado_con_obs") {
            toast.warning(`Aceptado con observaciones: ${data.sunat_mensaje || ""}`);
            notify("Aceptado con observaciones", `${numero}: ${data.sunat_mensaje || ""}`);
          } else if (data.estado === "rechazado") {
            toast.error(`SUNAT rechazó: ${data.sunat_codigo} ${data.sunat_mensaje || ""}`);
            notify("SUNAT rechazó", `${numero}: ${data.sunat_mensaje || ""}`);
          }
          // Auto-imprimir si está activado y SUNAT aceptó
          if (
            !autoPrintedRef.current &&
            (data.estado === "aceptado" || data.estado === "aceptado_con_obs") &&
            getPrinterSettings().auto &&
            getSavedPrinter()
          ) {
            autoPrintedRef.current = true;
            // setC primero para que imprimirTicket lea el comprobante actualizado
            setC(data);
            // Diferir un tick para que el setState surta efecto
            setTimeout(() => { void imprimirTicket(true); }, 50);
          }
        }
        lastEstado.current = data.estado;
      } catch {
        if (!cancelled) {
          toast.error("No encontrado");
          nav("/comprobantes", { replace: true });
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchOne();
    // Poll cada 2s mientras esté en proceso
    const tid = window.setInterval(() => {
      if (cancelled) return;
      if (lastEstado.current && ESTADOS_EN_PROCESO.has(lastEstado.current)) {
        fetchOne();
      }
    }, 2000);
    return () => {
      cancelled = true;
      window.clearInterval(tid);
    };
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

  async function verPDF() {
    if (!id) return;
    try {
      const blob = await apiBlob(`/api/v1/comprobantes/${id}/pdf`);
      const url = URL.createObjectURL(blob);
      window.open(url, "_blank");
      setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch {
      toast.error("No se pudo generar el PDF");
    }
  }

  async function imprimirTicket(silencioso = false): Promise<boolean> {
    if (!c) return false;
    const printer = getSavedPrinter();
    if (!printer) {
      if (!silencioso) {
        toast.error("Primero emparejá una impresora en Configuración → Impresora.");
      }
      return false;
    }
    const tenant = tenants.find((t) => t.id === user?.tenant_id);
    const { cols } = getPrinterSettings();
    setPrinting(true);
    try {
      const bytes = buildTicket(
        {
          ruc: tenant?.ruc || "",
          razon_social: tenant?.razon_social || "",
        },
        c,
        { cols, demoWatermark: !!tenant?.demo_mode },
      );
      await printBytes(bytes);
      if (!silencioso) toast.success("Ticket enviado a la impresora");
      return true;
    } catch (e: any) {
      if (!silencioso) toast.error(e?.message || "No se pudo imprimir");
      else toast.error("Auto-impresión falló: " + (e?.message || "error"));
      return false;
    } finally {
      setPrinting(false);
    }
  }

  async function reintentar() {
    if (!id) return;
    try {
      await api(`/api/v1/comprobantes/${id}/reintentar`, { method: "POST" });
      toast.success("Reencolado. Esperando respuesta de SUNAT…");
      // Forzar refresh inmediato; el polling tomará el cambio de estado.
      lastEstado.current = "pendiente";
      setC((cur) => (cur ? { ...cur, estado: "pendiente" } : cur));
    } catch (e: any) {
      toast.error("No se pudo reencolar: " + (e?.body?.detalle || e?.message || "error"));
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

      {ESTADOS_EN_PROCESO.has(c.estado) && (
        <div className="rounded-lg p-4 text-sm bg-slate-100 text-slate-700 border border-slate-200 flex items-center gap-3">
          <span className="inline-block h-2 w-2 rounded-full bg-amber-400 animate-pulse shrink-0" />
          <div className="flex-1">
            <div className="font-medium">
              {c.estado === "error" ? "Reintentando envío a SUNAT…" : "Enviando a SUNAT…"}
            </div>
            <div className="text-xs text-slate-500 mt-0.5">
              {c.estado === "error"
                ? "El sistema reintenta automáticamente con backoff exponencial. Si SUNAT ya volvió podés acelerar el siguiente intento."
                : "Esto se actualiza solo. SUNAT puede tardar entre 2 y 15 segundos."}
            </div>
            {c.estado === "error" && c.sunat_mensaje && (
              <div className="text-xs text-slate-500 mt-1 font-mono break-all">
                {c.sunat_mensaje}
              </div>
            )}
          </div>
          {c.estado === "error" && (
            <button onClick={reintentar} className="btn-ghost text-xs shrink-0">
              Reintentar ahora
            </button>
          )}
        </div>
      )}

      {!ESTADOS_EN_PROCESO.has(c.estado) && (c.sunat_codigo || c.sunat_mensaje) && (
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
            <button
              className="btn-primary w-full disabled:opacity-50"
              onClick={() => imprimirTicket()}
              disabled={printing || ESTADOS_EN_PROCESO.has(c.estado) ||
                (c.estado !== "aceptado" && c.estado !== "aceptado_con_obs")}
              title={detectCapability() === "unsupported"
                ? "Tu navegador no soporta Bluetooth; usá Chrome en Android o la app."
                : ""}
            >
              {printing ? "Imprimiendo…" : "🖨 Imprimir ticket"}
            </button>
            <button
              className="btn-ghost w-full disabled:opacity-50"
              onClick={() => verPDF()}
              disabled={ESTADOS_EN_PROCESO.has(c.estado)}
            >
              Ver PDF
            </button>
            <button
              className="btn-ghost w-full disabled:opacity-50"
              onClick={() => descargar("xml")}
              disabled={ESTADOS_EN_PROCESO.has(c.estado)}
            >
              Descargar XML firmado
            </button>
            <button
              className="btn-ghost w-full disabled:opacity-50"
              onClick={() => descargar("cdr")}
              disabled={ESTADOS_EN_PROCESO.has(c.estado)}
            >
              Descargar CDR (.zip)
            </button>
          </div>

          {(c.tipo === "01" || c.tipo === "03") &&
            (c.estado === "aceptado" || c.estado === "aceptado_con_obs") &&
            !c.anulado && (
              <div className="mt-6 pt-6 border-t border-slate-200 space-y-2">
                <p className="text-xs uppercase tracking-wide text-slate-500 font-medium">
                  Acciones sobre este comprobante
                </p>
                <Link to={`/notas/credito?ref=${c.id}`} className="btn-ghost w-full">
                  Nota de crédito
                </Link>
                <Link to={`/notas/debito?ref=${c.id}`} className="btn-ghost w-full">
                  Nota de débito
                </Link>
                <button onClick={() => setShowAnular(true)} className="btn-ghost w-full text-rose-600">
                  Anular (comunicación de baja)
                </button>
              </div>
            )}

          {c.anulado && (
            <div className="mt-6 pt-6 border-t border-slate-200">
              <div className="rounded-lg bg-rose-50 border border-rose-200 p-3 text-sm text-rose-800">
                Este comprobante fue anulado ante SUNAT.
              </div>
            </div>
          )}
        </div>
      </section>

      {showAnular && (
        <AnularModal
          comprobanteID={c.id}
          tipo={c.tipo}
          serieCorrelativo={`${c.serie}-${c.correlativo}`}
          onClose={() => setShowAnular(false)}
          onDone={(anulacionID) => {
            setShowAnular(false);
            nav(`/anulaciones/${anulacionID}`);
          }}
        />
      )}

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

function AnularModal({
  comprobanteID, tipo, serieCorrelativo, onClose, onDone,
}: {
  comprobanteID: string;
  tipo: string;
  serieCorrelativo: string;
  onClose: () => void;
  onDone: (anulacionID: string) => void;
}) {
  const [motivo, setMotivo] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (motivo.trim().length < 4) {
      toast.error("El motivo debe tener al menos 4 caracteres");
      return;
    }
    setSubmitting(true);
    try {
      const r = await api<{ id: string }>(
        `/api/v1/comprobantes/${comprobanteID}/anular`,
        { body: { motivo } }
      );
      toast.success("Anulación encolada, esperando respuesta de SUNAT…");
      onDone(r.id);
    } catch (e: any) {
      toast.error(e?.body?.detalle || "No se pudo anular");
    } finally {
      setSubmitting(false);
    }
  }

  const tipoLabel = tipo === "01" ? "factura" : tipo === "03" ? "boleta" : "comprobante";

  return (
    <Modal title={`Anular ${tipoLabel} ${serieCorrelativo}`} onClose={onClose}>
      <form onSubmit={onSubmit} className="space-y-4">
        <div className="rounded-lg bg-rose-50 border border-rose-200 p-3 text-sm text-rose-800">
          <strong>Atención:</strong> la comunicación de baja anula este comprobante
          ante SUNAT. Solo se puede hacer dentro de los 7 días siguientes a la
          emisión. Pasado ese plazo hay que emitir una nota de crédito en lugar
          de anular.
        </div>
        <div>
          <label className="label">Motivo de la anulación</label>
          <textarea
            className="input"
            rows={3}
            value={motivo}
            onChange={(e) => setMotivo(e.target.value)}
            placeholder="Ej: error en datos del cliente, operación no realizada, etc."
            required
            minLength={4}
            maxLength={120}
          />
          <p className="text-xs text-slate-400 mt-1">
            {motivo.length}/120 caracteres
          </p>
        </div>
        <div className="flex justify-end gap-2">
          <button type="button" className="btn-ghost" onClick={onClose}>
            Cancelar
          </button>
          <button type="submit" className="btn-danger" disabled={submitting}>
            {submitting ? "Enviando…" : "Confirmar anulación"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
