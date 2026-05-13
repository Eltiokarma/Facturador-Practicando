package http

import (
	"net/http"
	"time"

	"github.com/eltiokarma/facturador/apps/api/internal/sunatstatus"
)

type SunatStatusHandler struct {
	Checker *sunatstatus.Checker
}

type sunatStatusDTO struct {
	Beta sunatstatus.Status `json:"beta"`
	Prod sunatstatus.Status `json:"prod"`
	// AnyDown es true si alguno de los dos está caído. Útil para el banner
	// del frontend, que muestra una alerta general sin tener que conocer
	// los detalles.
	AnyDown bool      `json:"any_down"`
	Now     time.Time `json:"now"`
}

// Get devuelve el estado actual conocido de los endpoints de SUNAT.
// Endpoint público (no requiere auth) para que el banner del front lo
// pueda consultar incluso en el login.
func (h *SunatStatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	all := h.Checker.All()
	beta := all[sunatstatus.BillBeta]
	prod := all[sunatstatus.BillProd]
	writeJSON(w, http.StatusOK, sunatStatusDTO{
		Beta:    beta,
		Prod:    prod,
		AnyDown: !beta.Up || !prod.Up,
		Now:     time.Now().UTC(),
	})
}
