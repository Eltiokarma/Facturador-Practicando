package facturacion

import "testing"

func TestRecalcularTotales_FacturaGravada(t *testing.T) {
	f := Factura{
		Tipo:         "01",
		Serie:        "F001",
		Correlativo:  1,
		FechaEmision: "2026-05-12",
		Moneda:       "PEN",
		Receptor:     Receptor{TipoDoc: "6", NumDoc: "20000000001", RazonSocial: "X"},
		Items: []Item{
			{Descripcion: "A", Unidad: "NIU", Cantidad: 2, ValorUnitario: 50, AfectacionIGV: "10"},
			{Descripcion: "B", Unidad: "NIU", Cantidad: 1, ValorUnitario: 30, AfectacionIGV: "10"},
		},
	}
	if err := f.Validar(); err != nil {
		t.Fatalf("Validar: %v", err)
	}
	f.RecalcularTotales()

	if got, want := f.Totales.Gravado, 130.0; got != want {
		t.Errorf("Gravado = %v, want %v", got, want)
	}
	if got, want := f.Totales.IGV, 23.4; got != want {
		t.Errorf("IGV = %v, want %v", got, want)
	}
	if got, want := f.Totales.Total, 153.4; got != want {
		t.Errorf("Total = %v, want %v", got, want)
	}
	if got, want := f.Items[0].PrecioUnitario, 59.0; got != want {
		t.Errorf("PrecioUnitario[0] = %v, want %v", got, want)
	}
}

func TestValidar_RechazaFacturaConDNI(t *testing.T) {
	f := Factura{
		Tipo: "01", Serie: "F001", Correlativo: 1, FechaEmision: "2026-05-12", Moneda: "PEN",
		Receptor: Receptor{TipoDoc: "1", NumDoc: "12345678", RazonSocial: "X"},
		Items:    []Item{{Descripcion: "A", Cantidad: 1, ValorUnitario: 10, AfectacionIGV: "10"}},
	}
	if err := f.Validar(); err == nil {
		t.Fatal("se esperaba error: factura con DNI no es válida (debe ser RUC)")
	}
}

func TestValidar_BoletaSinReceptorAsumeDNI(t *testing.T) {
	f := Factura{
		Tipo: "03", Serie: "B001", Correlativo: 1, FechaEmision: "2026-05-12", Moneda: "PEN",
		Receptor: Receptor{NumDoc: "12345678", RazonSocial: "Cliente"},
		Items:    []Item{{Descripcion: "A", Cantidad: 1, ValorUnitario: 10, AfectacionIGV: "10"}},
	}
	if err := f.Validar(); err != nil {
		t.Fatalf("Validar boleta: %v", err)
	}
	if f.Receptor.TipoDoc != "1" {
		t.Errorf("se esperaba TipoDoc=1 por default en boleta, got %q", f.Receptor.TipoDoc)
	}
}
