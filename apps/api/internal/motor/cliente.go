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

type GuiaRequest struct {
	Modo   string         `json:"modo"`
	Tenant map[string]any `json:"tenant"`
	Guia   map[string]any `json:"guia"`
}

type GuiaResponse struct {
	Estado     string `json:"estado"` // "ticket" | "aceptado" | "rechazado" | "error"
	Ticket     string `json:"ticket,omitempty"`
	XMLFirmado string `json:"xml_firmado,omitempty"`
	CDRZip     string `json:"cdr_zip,omitempty"`
	Codigo     string `json:"codigo,omitempty"`
	Mensaje    string `json:"mensaje,omitempty"`
	HashCPE    string `json:"hash_cpe,omitempty"`
}

func (c *Cliente) Guia(ctx context.Context, req GuiaRequest) (*GuiaResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/emitir-guia", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("motor.Guia: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out GuiaResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("motor.Guia: %w (body=%q)", err, raw)
	}
	return &out, nil
}

type ConsultaTicketGuiaRequest struct {
	Modo   string         `json:"modo"`
	Tenant map[string]any `json:"tenant"`
	Ticket string         `json:"ticket"`
}

type ConsultaTicketGuiaResponse struct {
	// "procesando" | "aceptado" | "aceptado_con_obs" | "rechazado" | "error"
	Estado  string `json:"estado"`
	CDRZip  string `json:"cdr_zip,omitempty"`
	HashCPE string `json:"hash_cpe,omitempty"`
	Codigo  string `json:"codigo,omitempty"`
	Mensaje string `json:"mensaje,omitempty"`
}

func (c *Cliente) ConsultaTicketGuia(ctx context.Context, req ConsultaTicketGuiaRequest) (*ConsultaTicketGuiaResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/consulta-ticket-guia", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("motor.ConsultaTicketGuia: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out ConsultaTicketGuiaResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("motor.ConsultaTicketGuia: %w (body=%q)", err, raw)
	}
	return &out, nil
}

type AnularRequest struct {
	Modo      string         `json:"modo"`
	Tenant    map[string]any `json:"tenant"`
	Anulacion map[string]any `json:"anulacion"`
}

type AnularResponse struct {
	Estado     string `json:"estado"` // "ticket" | "error" | "rechazado"
	Ticket     string `json:"ticket,omitempty"`
	XMLFirmado string `json:"xml_firmado,omitempty"`
	Codigo     string `json:"codigo,omitempty"`
	Mensaje    string `json:"mensaje,omitempty"`
}

func (c *Cliente) Anular(ctx context.Context, req AnularRequest) (*AnularResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/anular", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("motor.Anular: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out AnularResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("motor.Anular: %w (body=%q)", err, raw)
	}
	return &out, nil
}

type ResumenRequest struct {
	Modo    string         `json:"modo"`
	Tenant  map[string]any `json:"tenant"`
	Resumen map[string]any `json:"resumen"`
}

type ResumenResponse struct {
	Estado      string `json:"estado"` // "ticket" | "error"
	Ticket      string `json:"ticket,omitempty"`
	XMLFirmado  string `json:"xml_firmado,omitempty"`
	Codigo      string `json:"codigo,omitempty"`
	Mensaje     string `json:"mensaje,omitempty"`
}

func (c *Cliente) Resumen(ctx context.Context, req ResumenRequest) (*ResumenResponse, error) {	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/resumen", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("motor.Resumen: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out ResumenResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("motor.Resumen: %w (body=%q)", err, raw)
	}
	return &out, nil
}

type ConsultaTicketRequest struct {
	Modo   string         `json:"modo"`
	Tenant map[string]any `json:"tenant"`
	Ticket string         `json:"ticket"`
}

type ConsultaTicketResponse struct {
	// "procesando" → SUNAT todavía no terminó, hay que volver a preguntar.
	// "aceptado", "aceptado_con_obs", "rechazado", "error"
	Estado  string `json:"estado"`
	CDRZip  string `json:"cdr_zip,omitempty"`
	Codigo  string `json:"codigo,omitempty"`
	Mensaje string `json:"mensaje,omitempty"`
}

func (c *Cliente) ConsultaTicket(ctx context.Context, req ConsultaTicketRequest) (*ConsultaTicketResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/consulta-ticket", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("motor.ConsultaTicket: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out ConsultaTicketResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("motor.ConsultaTicket: %w (body=%q)", err, raw)
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
