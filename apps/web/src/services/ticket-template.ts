import { EscPos } from "./escpos";
import { ComprobanteDetalleT } from "../types";

export type EmisorInfo = {
  ruc: string;
  razon_social: string;
  direccion_fiscal?: string;
  ubigeo?: string;
};

function tipoLabel(t: string): string {
  switch (t) {
    case "01": return "FACTURA ELECTRONICA";
    case "03": return "BOLETA DE VENTA ELECTRONICA";
    case "07": return "NOTA DE CREDITO ELECTRONICA";
    case "08": return "NOTA DE DEBITO ELECTRONICA";
    case "09": return "GUIA DE REMISION";
    default: return "COMPROBANTE";
  }
}

// Construye un ticket de 80mm (42 cols) listo para enviar a la impresora.
// Incluye QR con el formato SUNAT estándar.
export function buildTicket(
  emisor: EmisorInfo,
  c: ComprobanteDetalleT,
  opciones: { cols?: number; demoWatermark?: boolean } = {}
): Uint8Array {
  const cols = opciones.cols ?? 42;
  const p = new EscPos(cols).init();

  // ---- ENCABEZADO ----
  p.bold(true).size(2, 2).center(emisor.razon_social).size(1, 1).bold(false);
  p.center("RUC " + emisor.ruc);
  if (emisor.direccion_fiscal) p.center(emisor.direccion_fiscal);
  p.feed(1);

  if (opciones.demoWatermark) {
    p.center("*** DOCUMENTO DEMO — SIN VALOR LEGAL ***").feed(1);
  }

  // ---- TIPO Y NUMERO ----
  p.bold(true).center(tipoLabel(c.tipo)).bold(false);
  p.bold(true).size(2, 1).center(`${c.serie}-${c.correlativo}`).size(1, 1).bold(false);
  p.feed(1);

  // ---- DATOS GENERALES ----
  p.kv("Fecha emision:", c.fecha_emision);
  p.kv("Moneda:", c.moneda);
  p.hr();

  // ---- RECEPTOR ----
  p.bold(true).ln("CLIENTE").bold(false);
  p.ln(c.receptor_razon);
  const tipoDoc = c.receptor_tipo_doc === "6"
    ? "RUC"
    : c.receptor_tipo_doc === "1"
      ? "DNI"
      : "Doc.";
  p.kv(tipoDoc + ":", c.receptor_doc);
  p.hr();

  // ---- ITEMS ----
  const items = (c.payload?.items || []) as any[];
  for (const it of items) {
    p.ln(String(it.descripcion || ""));
    const qty = Number(it.cantidad || 0).toFixed(2);
    const vunit = Number(it.valor_unitario || 0).toFixed(2);
    const total = Number(it.total || 0).toFixed(2);
    p.kv(`  ${qty} ${it.unidad || "NIU"} x ${vunit}`, total);
  }
  p.hr();

  // ---- TOTALES ----
  if (c.gravado > 0)   p.kv("Op. Gravadas:",   `${c.moneda} ${c.gravado.toFixed(2)}`);
  if (c.exonerado > 0) p.kv("Op. Exoneradas:", `${c.moneda} ${c.exonerado.toFixed(2)}`);
  if (c.inafecto > 0)  p.kv("Op. Inafectas:",  `${c.moneda} ${c.inafecto.toFixed(2)}`);
  if (c.igv > 0)       p.kv("IGV (18%):",      `${c.moneda} ${c.igv.toFixed(2)}`);
  p.bold(true).size(1, 2).kv("TOTAL:", `${c.moneda} ${c.total.toFixed(2)}`).size(1, 1).bold(false);
  p.feed(1);

  // ---- QR ----
  const qrPayload = [
    emisor.ruc,
    c.tipo,
    c.serie,
    String(c.correlativo),
    c.igv.toFixed(2),
    c.total.toFixed(2),
    c.fecha_emision,
    c.receptor_tipo_doc,
    c.receptor_doc,
    c.hash_cpe || "",
  ].join("|");
  p.align("center");
  p.qr(qrPayload, 6);
  p.align("left");
  p.feed(1);

  // ---- FOOTER ----
  if (c.hash_cpe) {
    p.center("Hash CPE:");
    p.center(c.hash_cpe.length > cols ? c.hash_cpe.slice(0, cols) : c.hash_cpe);
  }
  if (c.estado) {
    p.center(`Estado SUNAT: ${c.estado}`);
  }
  if (c.anulado) {
    p.feed(1).bold(true).center("*** ANULADO ***").bold(false);
  }
  p.feed(3).cut();

  return p.build();
}
