package http

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/catalogos"
)

// ImportResultado es la respuesta de un import masivo: cuántas filas
// quedaron OK y, por cada fallida, en qué fila estaba y por qué.
type ImportResultado struct {
	Procesados int           `json:"procesados"`
	Creados    int           `json:"creados"`
	Errores    []ImportError `json:"errores"`
}

type ImportError struct {
	Fila    int    `json:"fila"`
	Mensaje string `json:"mensaje"`
}

// ImportarClientes recibe un archivo CSV multipart con columnas:
//   tipo_doc, num_doc, razon_social, direccion (opcional), email (opcional), telefono (opcional)
// La primera fila es el header.
func (h *ClientesHandler) ImportarCSV(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := auth.TenantIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	rows, err := parseCSVUpload(r, 4<<20)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "csv_invalido", "detalle": err.Error()})
		return
	}
	if len(rows) < 2 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "csv_vacio", "detalle": "se esperaba al menos un header y una fila de datos"})
		return
	}
	header := normalizeHeader(rows[0])
	cols := map[string]int{}
	for i, h := range header {
		cols[h] = i
	}
	required := []string{"tipo_doc", "num_doc", "razon_social"}
	for _, k := range required {
		if _, ok := cols[k]; !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "header_incompleto",
				"detalle": "falta columna obligatoria: " + k,
			})
			return
		}
	}

	res := ImportResultado{Procesados: len(rows) - 1}
	for i, row := range rows[1:] {
		fila := i + 2 // +1 por header, +1 por base 1
		c := catalogos.Cliente{
			TipoDoc:     getCell(row, cols, "tipo_doc"),
			NumDoc:      getCell(row, cols, "num_doc"),
			RazonSocial: getCell(row, cols, "razon_social"),
			Direccion:   getCell(row, cols, "direccion"),
			Email:       getCell(row, cols, "email"),
			Telefono:    getCell(row, cols, "telefono"),
		}
		if err := h.Store.Upsert(r.Context(), tenantID, &c); err != nil {
			res.Errores = append(res.Errores, ImportError{Fila: fila, Mensaje: err.Error()})
			continue
		}
		res.Creados++
	}
	writeJSON(w, http.StatusOK, res)
}

// ImportarCSV recibe columnas: codigo (opcional), descripcion, unidad,
// valor_unitario, afectacion_igv (default "10").
func (h *ProductosHandler) ImportarCSV(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := auth.TenantIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	rows, err := parseCSVUpload(r, 4<<20)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "csv_invalido", "detalle": err.Error()})
		return
	}
	if len(rows) < 2 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "csv_vacio"})
		return
	}
	header := normalizeHeader(rows[0])
	cols := map[string]int{}
	for i, h := range header {
		cols[h] = i
	}
	for _, k := range []string{"descripcion", "valor_unitario"} {
		if _, ok := cols[k]; !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "header_incompleto",
				"detalle": "falta columna obligatoria: " + k,
			})
			return
		}
	}

	res := ImportResultado{Procesados: len(rows) - 1}
	for i, row := range rows[1:] {
		fila := i + 2
		valStr := getCell(row, cols, "valor_unitario")
		val, err := strconv.ParseFloat(strings.ReplaceAll(valStr, ",", "."), 64)
		if err != nil {
			res.Errores = append(res.Errores, ImportError{
				Fila: fila, Mensaje: fmt.Sprintf("valor_unitario inválido: %q", valStr),
			})
			continue
		}
		afect := getCell(row, cols, "afectacion_igv")
		if afect == "" {
			afect = "10"
		}
		unidad := getCell(row, cols, "unidad")
		if unidad == "" {
			unidad = "NIU"
		}
		p := catalogos.Producto{
			Codigo:        getCell(row, cols, "codigo"),
			Descripcion:   getCell(row, cols, "descripcion"),
			Unidad:        unidad,
			ValorUnitario: val,
			AfectacionIGV: afect,
		}
		if err := h.Store.Upsert(r.Context(), tenantID, &p); err != nil {
			res.Errores = append(res.Errores, ImportError{Fila: fila, Mensaje: err.Error()})
			continue
		}
		res.Creados++
	}
	writeJSON(w, http.StatusOK, res)
}

// ---- helpers ----

func parseCSVUpload(r *http.Request, maxBytes int64) ([][]string, error) {
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		return nil, err
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("falta el campo 'file'")
	}
	defer file.Close()
	reader := csv.NewReader(io.LimitReader(file, maxBytes))
	reader.FieldsPerRecord = -1 // permitir filas con menos columnas (las tratamos como vacío)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// utf8BOM es el preámbulo que Excel agrega al guardar CSV.
var utf8BOM = string([]byte{0xEF, 0xBB, 0xBF})

func normalizeHeader(row []string) []string {
	out := make([]string, len(row))
	for i, c := range row {
		c = strings.TrimPrefix(c, utf8BOM)
		out[i] = strings.ToLower(strings.TrimSpace(c))
	}
	return out
}

func getCell(row []string, cols map[string]int, key string) string {
	if idx, ok := cols[key]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}
