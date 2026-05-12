// Package pdf genera la representación impresa (PDF A4) de un comprobante
// electrónico. Incluye QR con el formato SUNAT:
//
//	RUC|TIPO|SERIE|CORRELATIVO|IGV|TOTAL|FECHA|TIPO_DOC_REC|NUM_DOC_REC|HASH
//
// La idea no es ser bonito en exceso — es ser legible, profesional y caber
// en una A4 sin recortes.
package pdf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"

	"github.com/go-pdf/fpdf"
	"github.com/skip2/go-qrcode"

	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

// Builder es stateless. Cada llamada a Render() recibe el tenant emisor.
type Builder struct{}

func New() *Builder { return &Builder{} }

func labelTipo(t string) string {
	switch t {
	case "01":
		return "FACTURA ELECTRÓNICA"
	case "03":
		return "BOLETA DE VENTA ELECTRÓNICA"
	case "07":
		return "NOTA DE CRÉDITO ELECTRÓNICA"
	case "08":
		return "NOTA DE DÉBITO ELECTRÓNICA"
	default:
		return "COMPROBANTE ELECTRÓNICO"
	}
}

// Render produce el PDF en bytes a partir del detalle del comprobante
// emitido por el tenant t.
func (b *Builder) Render(t *tenant.Record, d *comprobantes.Detalle) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// ----------------- ENCABEZADO -----------------
	// Columna izquierda: datos del emisor
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(115, 6, tr(t.RazonSocial), "", 2, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(115, 5, tr("RUC "+t.RUC), "", 2, "L", false, 0, "")
	pdf.CellFormat(115, 5, tr(t.DireccionFiscal), "", 2, "L", false, 0, "")
	if t.NombreComercial != "" && t.NombreComercial != t.RazonSocial {
		pdf.CellFormat(115, 5, tr(t.NombreComercial), "", 2, "L", false, 0, "")
	}

	// Columna derecha: caja con tipo + número
	pdf.SetXY(135, 15)
	pdf.SetDrawColor(99, 102, 241)
	pdf.SetLineWidth(0.5)
	pdf.Rect(135, 15, 60, 25, "D")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(99, 102, 241)
	pdf.SetXY(135, 18)
	pdf.CellFormat(60, 5, tr(labelTipo(d.Tipo)), "", 0, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(71, 85, 105)
	pdf.SetXY(135, 24)
	pdf.CellFormat(60, 4, tr("RUC "+t.RUC), "", 0, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetXY(135, 29)
	pdf.CellFormat(60, 8, fmt.Sprintf("%s-%d", d.Serie, d.Correlativo), "", 0, "C", false, 0, "")

	pdf.SetY(48)

	// ----------------- RECEPTOR -----------------
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetLineWidth(0.2)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(40, 5, tr("Cliente:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(140, 5, tr(d.ReceptorRazon), "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(40, 5, tr("Documento:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(140, 5, tr(tipoDocLabel(d.ReceptorTipoDoc)+" "+d.ReceptorDoc), "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(40, 5, tr("Fecha de emisión:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(140, 5, d.FechaEmision, "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(40, 5, tr("Moneda:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(140, 5, d.Moneda, "", 1, "L", false, 0, "")

	pdf.Ln(3)

	// ----------------- ÍTEMS -----------------
	items := extractItems(d.Payload)
	pdf.SetFillColor(15, 23, 42)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.CellFormat(15, 7, tr("Cant."), "", 0, "C", true, 0, "")
	pdf.CellFormat(20, 7, tr("Unidad"), "", 0, "C", true, 0, "")
	pdf.CellFormat(90, 7, tr("Descripción"), "", 0, "L", true, 0, "")
	pdf.CellFormat(25, 7, tr("V. Unit"), "", 0, "R", true, 0, "")
	pdf.CellFormat(30, 7, tr("Total"), "", 1, "R", true, 0, "")

	pdf.SetTextColor(15, 23, 42)
	pdf.SetFont("Helvetica", "", 8)
	alt := false
	for _, it := range items {
		if alt {
			pdf.SetFillColor(248, 250, 252)
			pdf.SetTextColor(15, 23, 42)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		alt = !alt
		pdf.CellFormat(15, 6, fmt.Sprintf("%.2f", it.Cantidad), "", 0, "R", true, 0, "")
		pdf.CellFormat(20, 6, tr(it.Unidad), "", 0, "C", true, 0, "")
		pdf.CellFormat(90, 6, tr(trunc(it.Descripcion, 55)), "", 0, "L", true, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", it.ValorUnitario), "", 0, "R", true, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.2f", it.Total), "", 1, "R", true, 0, "")
	}

	pdf.Ln(4)

	// ----------------- TOTALES -----------------
	startY := pdf.GetY()
	pdf.SetXY(120, startY)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(71, 85, 105)

	addTotal := func(label string, val float64) {
		pdf.SetX(120)
		pdf.CellFormat(40, 5, tr(label), "", 0, "L", false, 0, "")
		pdf.CellFormat(35, 5, fmt.Sprintf("%s %.2f", d.Moneda, val), "", 1, "R", false, 0, "")
	}
	if d.Gravado > 0 {
		addTotal("Gravado:", d.Gravado)
	}
	if d.Exonerado > 0 {
		addTotal("Exonerado:", d.Exonerado)
	}
	if d.Inafecto > 0 {
		addTotal("Inafecto:", d.Inafecto)
	}
	if d.IGV > 0 {
		addTotal("IGV (18%):", d.IGV)
	}

	pdf.SetX(120)
	pdf.SetDrawColor(15, 23, 42)
	pdf.SetLineWidth(0.4)
	pdf.Line(120, pdf.GetY()+1, 195, pdf.GetY()+1)
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetX(120)
	pdf.CellFormat(40, 8, tr("TOTAL:"), "", 0, "L", false, 0, "")
	pdf.CellFormat(35, 8, fmt.Sprintf("%s %.2f", d.Moneda, d.Total), "", 1, "R", false, 0, "")

	// Monto en letras
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(71, 85, 105)
	pdf.SetXY(15, pdf.GetY()+4)
	pdf.MultiCell(105, 4, tr("Son: "+montoEnLetras(d.Total, d.Moneda)), "", "L", false)

	// ----------------- QR + HASH -----------------
	qrPayload := fmt.Sprintf("%s|%s|%s|%d|%.2f|%.2f|%s|%s|%s|%s",
		t.RUC, d.Tipo, d.Serie, d.Correlativo,
		d.IGV, d.Total, d.FechaEmision,
		d.ReceptorTipoDoc, d.ReceptorDoc, d.HashCPE,
	)
	if pngBytes, err := qrcode.Encode(qrPayload, qrcode.Medium, 256); err == nil {
		if img, err := png.Decode(bytes.NewReader(pngBytes)); err == nil {
			imgName := "qr_" + d.ID.String()
			pdf.RegisterImageOptionsReader(imgName, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, encodePNG(img))
			pdf.ImageOptions(imgName, 15, pdf.GetY()+4, 35, 35, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		}
	}

	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(100, 116, 139)
	pdf.SetXY(55, pdf.GetY()+8)
	pdf.MultiCell(140, 3.5, tr("Representación impresa de la "+labelTipo(d.Tipo)+
		". Verificá en SUNAT con el código QR o consultando con el hash y los datos del comprobante."), "", "L", false)
	pdf.SetXY(55, pdf.GetY()+1)
	if d.HashCPE != "" {
		pdf.CellFormat(140, 4, tr("Hash CPE: "+trunc(d.HashCPE, 60)), "", 1, "L", false, 0, "")
	}
	if d.Estado != "" {
		pdf.SetX(55)
		pdf.CellFormat(140, 4, tr("Estado SUNAT: "+d.Estado+" "+d.SunatCodigo+" "+d.SunatMensaje), "", 1, "L", false, 0, "")
	}

	// Pie
	pdf.SetY(-15)
	pdf.SetFont("Helvetica", "I", 7)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(0, 4, tr("Generado por Facturador Self-Hosted Perú"), "", 0, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type itemView struct {
	Descripcion   string
	Unidad        string
	Cantidad      float64
	ValorUnitario float64
	Total         float64
}

func extractItems(payload json.RawMessage) []itemView {
	var raw struct {
		Items []struct {
			Descripcion   string  `json:"descripcion"`
			Unidad        string  `json:"unidad"`
			Cantidad      float64 `json:"cantidad"`
			ValorUnitario float64 `json:"valor_unitario"`
			Total         float64 `json:"total"`
		} `json:"items"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil
	}
	out := make([]itemView, len(raw.Items))
	for i, it := range raw.Items {
		out[i] = itemView{
			Descripcion:   it.Descripcion,
			Unidad:        it.Unidad,
			Cantidad:      it.Cantidad,
			ValorUnitario: it.ValorUnitario,
			Total:         it.Total,
		}
	}
	return out
}

func encodePNG(img image.Image) *bytes.Reader {
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return bytes.NewReader(b.Bytes())
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// tr convierte UTF-8 a la codificación que usa fpdf por defecto (cp1252).
func tr(s string) string {
	return string([]byte(toCp1252(s)))
}

func toCp1252(s string) string {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r < 0x80:
			out = append(out, byte(r))
		case r >= 0xA0 && r <= 0xFF:
			out = append(out, byte(r))
		default:
			// Mapeo mínimo de caracteres comunes en español/portugués
			switch r {
			case 'á':
				out = append(out, 0xE1)
			case 'é':
				out = append(out, 0xE9)
			case 'í':
				out = append(out, 0xED)
			case 'ó':
				out = append(out, 0xF3)
			case 'ú':
				out = append(out, 0xFA)
			case 'Á':
				out = append(out, 0xC1)
			case 'É':
				out = append(out, 0xC9)
			case 'Í':
				out = append(out, 0xCD)
			case 'Ó':
				out = append(out, 0xD3)
			case 'Ú':
				out = append(out, 0xDA)
			case 'ñ':
				out = append(out, 0xF1)
			case 'Ñ':
				out = append(out, 0xD1)
			case 'ü':
				out = append(out, 0xFC)
			case 'Ü':
				out = append(out, 0xDC)
			case '¿':
				out = append(out, 0xBF)
			case '¡':
				out = append(out, 0xA1)
			case '°':
				out = append(out, 0xB0)
			case '€':
				out = append(out, 0x80)
			case '…':
				out = append(out, '.', '.', '.')
			default:
				out = append(out, '?')
			}
		}
	}
	return string(out)
}

func tipoDocLabel(t string) string {
	switch t {
	case "1":
		return "DNI"
	case "4":
		return "Carnet Ext."
	case "6":
		return "RUC"
	case "7":
		return "Pasaporte"
	case "0":
		return "Sin doc."
	default:
		return "Doc."
	}
}

// montoEnLetras: simplificación — un proyecto serio metería num2letters
// completo. Por ahora cubre el común.
func montoEnLetras(monto float64, moneda string) string {
	entero := int64(monto)
	centavos := int64((monto - float64(entero)) * 100)
	unidad := "SOLES"
	if moneda == "USD" {
		unidad = "DÓLARES AMERICANOS"
	}
	return fmt.Sprintf("%s CON %02d/100 %s", spell(entero), centavos, unidad)
}
