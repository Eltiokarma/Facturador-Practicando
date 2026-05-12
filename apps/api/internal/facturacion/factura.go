// Package facturacion define el modelo de comprobante y la lógica de
// cálculo de totales. Los totales SIEMPRE se recalculan server-side;
// los del cliente son referencia, nunca verdad.
package facturacion

import (
	"errors"
	"fmt"
	"math"
)

type Receptor struct {
	TipoDoc     string `json:"tipo_doc"`
	NumDoc      string `json:"num_doc"`
	RazonSocial string `json:"razon_social"`
	Direccion   string `json:"direccion,omitempty"`
}

type Item struct {
	Codigo         string  `json:"codigo,omitempty"`
	Descripcion    string  `json:"descripcion"`
	Unidad         string  `json:"unidad"` // catálogo 03 SUNAT, default "NIU"
	Cantidad       float64 `json:"cantidad"`
	ValorUnitario  float64 `json:"valor_unitario"`  // sin IGV
	AfectacionIGV  string  `json:"afectacion_igv"`  // catálogo 07: "10" gravado, "20" exonerado, "30" inafecto
	PorcentajeIGV  float64 `json:"porcentaje_igv,omitempty"` // default 18 si afectacion=10
	// Campos calculados (no se confía en lo que mande el cliente):
	IGV             float64 `json:"igv,omitempty"`
	PrecioUnitario  float64 `json:"precio_unitario,omitempty"`
	Total           float64 `json:"total,omitempty"`
}

type Totales struct {
	Gravado    float64 `json:"gravado"`
	Exonerado  float64 `json:"exonerado"`
	Inafecto   float64 `json:"inafecto"`
	Gratuito   float64 `json:"gratuito"`
	IGV        float64 `json:"igv"`
	ISC        float64 `json:"isc"`
	ICBPER     float64 `json:"icbper"`
	Total      float64 `json:"total"`
}

type Factura struct {
	Tipo          string   `json:"tipo"`         // "01" factura, "03" boleta
	Serie         string   `json:"serie"`        // "F001" o "B001"
	Correlativo   int64    `json:"correlativo"`
	FechaEmision  string   `json:"fecha_emision"` // YYYY-MM-DD
	Moneda        string   `json:"moneda"`        // PEN, USD
	TipoOperacion string   `json:"tipo_operacion,omitempty"` // default "0101"
	Receptor      Receptor `json:"receptor"`
	Items         []Item   `json:"items"`
	Totales       Totales  `json:"totales,omitempty"` // se sobrescribe server-side
}

// Validar revisa estructura mínima y reglas SUNAT básicas.
func (f *Factura) Validar() error {
	if f.Tipo != "01" && f.Tipo != "03" {
		return fmt.Errorf("tipo de comprobante no soportado en MVP: %q (esperaba 01=factura o 03=boleta)", f.Tipo)
	}
	if len(f.Serie) != 4 {
		return fmt.Errorf("serie inválida: %q (debe ser 4 caracteres)", f.Serie)
	}
	if f.Tipo == "01" && f.Serie[0] != 'F' {
		return errors.New("factura debe tener serie que empiece con 'F'")
	}
	if f.Tipo == "03" && f.Serie[0] != 'B' {
		return errors.New("boleta debe tener serie que empiece con 'B'")
	}
	if f.Correlativo <= 0 {
		return errors.New("correlativo debe ser > 0")
	}
	if f.FechaEmision == "" {
		return errors.New("fecha_emision requerida")
	}
	if f.Moneda == "" {
		return errors.New("moneda requerida")
	}
	if len(f.Items) == 0 {
		return errors.New("la factura no tiene ítems")
	}
	switch f.Tipo {
	case "01": // factura → receptor con RUC
		if f.Receptor.TipoDoc != "6" {
			return errors.New("factura requiere receptor con RUC (tipo_doc=6)")
		}
		if len(f.Receptor.NumDoc) != 11 {
			return fmt.Errorf("RUC del receptor inválido: %q", f.Receptor.NumDoc)
		}
	case "03": // boleta → DNI u otro
		if f.Receptor.TipoDoc == "" {
			f.Receptor.TipoDoc = "1" // DNI por defecto
		}
	}
	if f.Receptor.RazonSocial == "" {
		return errors.New("razon_social del receptor requerida")
	}
	for i := range f.Items {
		it := &f.Items[i]
		if it.Cantidad <= 0 {
			return fmt.Errorf("ítem %d: cantidad debe ser > 0", i+1)
		}
		if it.ValorUnitario < 0 {
			return fmt.Errorf("ítem %d: valor_unitario no puede ser negativo", i+1)
		}
		if it.AfectacionIGV == "" {
			it.AfectacionIGV = "10"
		}
		if it.Unidad == "" {
			it.Unidad = "NIU"
		}
	}
	return nil
}

// RecalcularTotales recompone IGV e importes ignorando lo que vino del cliente.
// Convención: valor_unitario es sin IGV. Para afectación 10 se aplica 18% IGV.
// Para 20 (exonerado) y 30 (inafecto), IGV = 0.
func (f *Factura) RecalcularTotales() {
	var grav, exo, ina, gra, igvTotal float64
	for i := range f.Items {
		it := &f.Items[i]
		subtotal := round2(it.ValorUnitario * it.Cantidad)
		var igv float64
		var precioUnit float64
		switch it.AfectacionIGV {
		case "10": // gravado
			if it.PorcentajeIGV == 0 {
				it.PorcentajeIGV = 18
			}
			igv = round2(subtotal * (it.PorcentajeIGV / 100))
			precioUnit = round2(it.ValorUnitario * (1 + it.PorcentajeIGV/100))
			grav += subtotal
			igvTotal += igv
		case "20": // exonerado
			precioUnit = it.ValorUnitario
			exo += subtotal
		case "30": // inafecto
			precioUnit = it.ValorUnitario
			ina += subtotal
		default: // gratuitas y otros: tratar como inafecto del total (simplificado MVP)
			precioUnit = it.ValorUnitario
			gra += subtotal
		}
		it.IGV = igv
		it.PrecioUnitario = precioUnit
		it.Total = round2(subtotal + igv)
	}
	f.Totales = Totales{
		Gravado:   round2(grav),
		Exonerado: round2(exo),
		Inafecto:  round2(ina),
		Gratuito:  round2(gra),
		IGV:       round2(igvTotal),
		Total:     round2(grav + exo + ina + igvTotal),
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
