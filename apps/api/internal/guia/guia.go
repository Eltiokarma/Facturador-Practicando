// Package guia define el modelo de Guía de Remisión Electrónica (GRE)
// y las reglas SUNAT específicas.
//
// La GRE acompaña el traslado físico de mercadería. SUNAT la exige
// obligatoriamente desde julio 2026. Es tipo de documento '09' en el
// catálogo SUNAT 01.
//
// Diferencias clave con factura/boleta:
//   - Endpoint SUNAT distinto: api-cpe.sunat.gob.pe (REST + OAuth2),
//     no el SOAP de facturas.
//   - No lleva totales monetarios — solo descripción de ítems trasladados.
//   - Lleva direcciones de partida y llegada con ubigeo de SUNAT.
//   - Lleva datos del transporte (privado o público).
//   - Lleva peso bruto total en KGM.
package guia

import (
	"errors"
	"fmt"
	"regexp"
)

// Punto representa una dirección con ubigeo.
type Punto struct {
	Ubigeo    string `json:"ubigeo"`     // 6 dígitos catálogo SUNAT 13
	Direccion string `json:"direccion"`
}

// Destinatario es el receptor del traslado.
type Destinatario struct {
	TipoDoc     string `json:"tipo_doc"`     // 1 DNI, 6 RUC, etc.
	NumDoc      string `json:"num_doc"`
	RazonSocial string `json:"razon_social"`
}

// ModalidadTransporte: catálogo SUNAT 18.
//   01 = Transporte público (lo hace un tercero con RUC).
//   02 = Transporte privado (lo hace el propio emisor).
type ModalidadTransporte string

const (
	TransportePublico  ModalidadTransporte = "01"
	TransportePrivado  ModalidadTransporte = "02"
)

// MotivoTraslado: catálogo SUNAT 20.
//   01 = Venta
//   02 = Compra
//   04 = Traslado entre establecimientos de la misma empresa
//   13 = Otros
type MotivoTraslado string

type Transportista struct {
	RUC         string `json:"ruc,omitempty"`           // solo si modalidad=01
	RazonSocial string `json:"razon_social,omitempty"`
	PlacaVehiculo string `json:"placa_vehiculo,omitempty"` // 6-8 caracteres
	DocChofer   string `json:"doc_chofer,omitempty"`     // DNI/CE del chofer
	NombreChofer string `json:"nombre_chofer,omitempty"`
	LicenciaChofer string `json:"licencia_chofer,omitempty"`
}

type ItemTraslado struct {
	Codigo      string  `json:"codigo,omitempty"`
	Descripcion string  `json:"descripcion"`
	Unidad      string  `json:"unidad"`   // catálogo SUNAT 03 (NIU, KGM, etc.)
	Cantidad    float64 `json:"cantidad"`
}

type Guia struct {
	Serie         string              `json:"serie"`        // T###
	Correlativo   int64               `json:"correlativo"`
	FechaEmision  string              `json:"fecha_emision"` // YYYY-MM-DD
	FechaTraslado string              `json:"fecha_traslado"` // YYYY-MM-DD, cuándo arranca el transporte
	Destinatario  Destinatario        `json:"destinatario"`
	PuntoPartida  Punto               `json:"punto_partida"`
	PuntoLlegada  Punto               `json:"punto_llegada"`
	Motivo        MotivoTraslado      `json:"motivo"`
	Modalidad     ModalidadTransporte `json:"modalidad"`
	PesoTotalKg   float64             `json:"peso_total_kg"`
	NumBultos     int                 `json:"num_bultos,omitempty"`
	Transportista *Transportista      `json:"transportista,omitempty"`
	Items         []ItemTraslado      `json:"items"`
	// Comprobante relacionado (opcional, ej. factura de la venta que se está trasladando)
	DocRelacionadoTipo string `json:"doc_relacionado_tipo,omitempty"` // 01 factura
	DocRelacionadoSerie string `json:"doc_relacionado_serie,omitempty"`
	DocRelacionadoNum   int64  `json:"doc_relacionado_num,omitempty"`
	Observaciones string `json:"observaciones,omitempty"`
}

var (
	ubigeoRe  = regexp.MustCompile(`^\d{6}$`)
	rucRe     = regexp.MustCompile(`^\d{11}$`)
	dniRe     = regexp.MustCompile(`^\d{8}$`)
	serieGRE  = regexp.MustCompile(`^T\d{3}$`)
	placaRe   = regexp.MustCompile(`^[A-Z0-9-]{6,8}$`)
)

func (g *Guia) Validar() error {
	if !serieGRE.MatchString(g.Serie) {
		return fmt.Errorf("serie de GRE inválida: %q (debe ser T### — T seguido de 3 dígitos)", g.Serie)
	}
	if g.Correlativo <= 0 {
		return errors.New("correlativo debe ser > 0")
	}
	if g.FechaEmision == "" {
		return errors.New("fecha_emision requerida")
	}
	if g.FechaTraslado == "" {
		return errors.New("fecha_traslado requerida")
	}

	// Destinatario
	switch g.Destinatario.TipoDoc {
	case "1":
		if !dniRe.MatchString(g.Destinatario.NumDoc) {
			return fmt.Errorf("DNI del destinatario inválido: %q", g.Destinatario.NumDoc)
		}
	case "6":
		if !rucRe.MatchString(g.Destinatario.NumDoc) {
			return fmt.Errorf("RUC del destinatario inválido: %q", g.Destinatario.NumDoc)
		}
	case "4", "7", "0":
		// CE, pasaporte, sin doc — no validamos longitud
	default:
		return fmt.Errorf("destinatario.tipo_doc inválido: %q", g.Destinatario.TipoDoc)
	}
	if g.Destinatario.RazonSocial == "" {
		return errors.New("destinatario.razon_social requerida")
	}

	// Puntos
	if !ubigeoRe.MatchString(g.PuntoPartida.Ubigeo) {
		return fmt.Errorf("ubigeo de partida inválido: %q (esperaba 6 dígitos)", g.PuntoPartida.Ubigeo)
	}
	if g.PuntoPartida.Direccion == "" {
		return errors.New("direccion de partida requerida")
	}
	if !ubigeoRe.MatchString(g.PuntoLlegada.Ubigeo) {
		return fmt.Errorf("ubigeo de llegada inválido: %q", g.PuntoLlegada.Ubigeo)
	}
	if g.PuntoLlegada.Direccion == "" {
		return errors.New("direccion de llegada requerida")
	}

	// Motivo y modalidad
	if g.Motivo == "" {
		return errors.New("motivo de traslado requerido (catálogo SUNAT 20)")
	}
	if g.Modalidad != TransportePublico && g.Modalidad != TransportePrivado {
		return fmt.Errorf("modalidad inválida: %q (esperaba 01 público o 02 privado)", g.Modalidad)
	}

	// Transporte
	if g.Modalidad == TransportePublico {
		if g.Transportista == nil || !rucRe.MatchString(g.Transportista.RUC) {
			return errors.New("transporte público requiere RUC del transportista")
		}
		if g.Transportista.RazonSocial == "" {
			return errors.New("transporte público requiere razón social del transportista")
		}
	} else {
		// privado: requiere placa + datos del chofer
		if g.Transportista == nil {
			return errors.New("transporte privado requiere datos del vehículo y chofer")
		}
		if !placaRe.MatchString(g.Transportista.PlacaVehiculo) {
			return fmt.Errorf("placa de vehículo inválida: %q", g.Transportista.PlacaVehiculo)
		}
		if g.Transportista.DocChofer == "" || g.Transportista.NombreChofer == "" {
			return errors.New("transporte privado requiere documento y nombre del chofer")
		}
	}

	if g.PesoTotalKg <= 0 {
		return errors.New("peso_total_kg debe ser > 0")
	}
	if len(g.Items) == 0 {
		return errors.New("la guía no tiene ítems")
	}
	for i, it := range g.Items {
		if it.Descripcion == "" {
			return fmt.Errorf("ítem %d: descripción requerida", i+1)
		}
		if it.Cantidad <= 0 {
			return fmt.Errorf("ítem %d: cantidad debe ser > 0", i+1)
		}
		if it.Unidad == "" {
			g.Items[i].Unidad = "NIU"
		}
	}
	return nil
}
