package sunat

import "github.com/eltiokarma/facturador/apps/api/internal/config"

// Endpoints SOAP de SUNAT por modo.
type Endpoints struct {
	BillService        string // facturas, boletas, notas
	GuiaService        string // GRE (v2)
	ConsultBillService string // consulta de CDR
}

func For(mode config.SunatMode) Endpoints {
	switch mode {
	case config.SunatModeProd:
		return Endpoints{
			BillService:        "https://e-factura.sunat.gob.pe/ol-ti-itcpfegem/billService",
			GuiaService:        "https://e-guiaremision.sunat.gob.pe/ol-ti-itemision-guia-gem/billService",
			ConsultBillService: "https://e-factura.sunat.gob.pe/ol-it-wsconscpegem/billConsultService",
		}
	default: // beta
		return Endpoints{
			BillService:        "https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService",
			GuiaService:        "https://e-beta.sunat.gob.pe/ol-ti-itemision-guia-gem-beta/billService",
			ConsultBillService: "https://e-beta.sunat.gob.pe/ol-it-wsconscpegem-beta/billConsultService",
		}
	}
}
