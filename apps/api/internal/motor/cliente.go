// Package motor habla con el microservicio PHP/Greenter via HTTP.
// El contrato vive en apps/motor/CONTRACT.md.
package motor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Cliente struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Cliente {
	return &Cliente{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 60 * time.Second, // SUNAT a veces tarda
		},
	}
}

type EmitirRequest struct {
	Modo        string                 `json:"modo"`
	Tenant      map[string]any         `json:"tenant"`
	Comprobante map[string]any         `json:"comprobante"`
}

type EmitirResponse struct {
	Estado        string   `json:"estado"`
	XMLFirmado    string   `json:"xml_firmado,omitempty"`
	CDRZip        string   `json:"cdr_zip,omitempty"`
	HashCPE       string   `json:"hash_cpe,omitempty"`
	Codigo        string   `json:"codigo,omitempty"`
	Mensaje       string   `json:"mensaje,omitempty"`
	Observaciones []string `json:"observaciones,omitempty"`
}

func (c *Cliente) Emitir(ctx context.Context, req EmitirRequest) (*EmitirResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("motor: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/emitir", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("motor: HTTP: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("motor: leer body: %w", err)
	}
	var out EmitirResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("motor: respuesta no es JSON válido (status=%d, body=%q): %w", resp.StatusCode, string(raw), err)
	}
	return &out, nil
}

func (c *Cliente) Health(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("motor health: status %d", resp.StatusCode)
	}
	return nil
}
