// Package anulaciones gestiona las "comunicaciones de baja" — el documento
// UBL VoidedDocuments que SUNAT exige para anular un CPE ya aceptado.
//
// SUNAT permite anular dentro de los 7 días siguientes a la fecha de emisión.
// Pasado ese plazo, hay que emitir una nota de crédito en lugar de anular.
package anulaciones

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(p *pgxpool.Pool) *Store { return &Store{pool: p} }

var (
	ErrNotFound      = errors.New("anulaciones: no encontrado")
	ErrFueraPlazo    = errors.New("anulaciones: comprobante con más de 7 días — emití nota de crédito")
	ErrYaAnulado     = errors.New("anulaciones: el comprobante ya fue anulado")
	ErrNoAceptado    = errors.New("anulaciones: solo se anulan comprobantes aceptados")
)

type ComprobanteAnular struct {
	ID            uuid.UUID `json:"id"`
	Tipo          string    `json:"tipo"`
	Serie         string    `json:"serie"`
	Correlativo   int64     `json:"correlativo"`
	FechaEmision  time.Time `json:"-"`
	ReceptorRazon string    `json:"receptor_razon"`
	Total         float64   `json:"total"`
	Moneda        string    `json:"moneda"`
	Motivo        string    `json:"motivo"`
}

type Anulacion struct {
	ID                  uuid.UUID `json:"id"`
	Serie               string    `json:"serie"`
	Correlativo         int       `json:"correlativo"`
	FechaReferencia     string    `json:"fecha_referencia"`
	FechaEmision        string    `json:"fecha_emision"`
	Estado              string    `json:"estado"`
	Ticket              string    `json:"ticket,omitempty"`
	SunatCodigo         string    `json:"sunat_codigo,omitempty"`
	SunatMensaje        string    `json:"sunat_mensaje,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// CrearIndividual crea una comunicación de baja para UN comprobante.
// Valida que esté aceptado, no anulado, y dentro del plazo de 7 días.
// Devuelve también la fecha de referencia (la de emisión del comprobante).
func (s *Store) CrearIndividual(ctx context.Context, tenantID, comprobanteID uuid.UUID, motivo string) (*Anulacion, error) {
	if motivo == "" {
		return nil, errors.New("motivo requerido")
	}

	// Validar el comprobante a anular en una sola consulta.
	var (
		estado        string
		fechaEmision  time.Time
		anulado       bool
	)
	err := s.pool.QueryRow(ctx, `
		SELECT estado, fecha_emision, anulado
		FROM comprobantes WHERE tenant_id=$1 AND id=$2
	`, tenantID, comprobanteID).Scan(&estado, &fechaEmision, &anulado)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if anulado {
		return nil, ErrYaAnulado
	}
	if estado != "aceptado" && estado != "aceptado_con_obs" {
		return nil, ErrNoAceptado
	}
	if time.Since(fechaEmision) > 7*24*time.Hour {
		return nil, ErrFueraPlazo
	}

	serie := "RA-" + fechaEmision.Format("20060102")
	hoy := time.Now().UTC().Truncate(24 * time.Hour)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var correlativo int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(correlativo),0)+1
		FROM anulaciones WHERE tenant_id=$1 AND serie=$2
	`, tenantID, serie).Scan(&correlativo); err != nil {
		return nil, err
	}
	id := uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO anulaciones (id, tenant_id, serie, correlativo, fecha_referencia, fecha_emision, estado)
		VALUES ($1,$2,$3,$4,$5,$6,'pendiente')
	`, id, tenantID, serie, correlativo, fechaEmision, hoy); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO anulacion_items (anulacion_id, comprobante_id, motivo)
		VALUES ($1,$2,$3)
	`, id, comprobanteID, motivo); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID, id)
}

func (s *Store) Get(ctx context.Context, tenantID, id uuid.UUID) (*Anulacion, error) {
	var a Anulacion
	var fechaRef, fechaEmi time.Time
	var ticket, code, msg *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, serie, correlativo, fecha_referencia, fecha_emision,
		       estado, ticket, sunat_codigo, sunat_mensaje, created_at
		FROM anulaciones WHERE tenant_id=$1 AND id=$2
	`, tenantID, id).Scan(&a.ID, &a.Serie, &a.Correlativo, &fechaRef, &fechaEmi,
		&a.Estado, &ticket, &code, &msg, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.FechaReferencia = fechaRef.Format("2006-01-02")
	a.FechaEmision = fechaEmi.Format("2006-01-02")
	if ticket != nil {
		a.Ticket = *ticket
	}
	if code != nil {
		a.SunatCodigo = *code
	}
	if msg != nil {
		a.SunatMensaje = *msg
	}
	return &a, nil
}

func (s *Store) List(ctx context.Context, tenantID uuid.UUID, limit int) ([]Anulacion, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, serie, correlativo, fecha_referencia, fecha_emision,
		       estado, COALESCE(ticket,''), COALESCE(sunat_codigo,''), COALESCE(sunat_mensaje,''),
		       created_at
		FROM anulaciones WHERE tenant_id=$1
		ORDER BY created_at DESC LIMIT %d
	`, limit), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Anulacion
	for rows.Next() {
		var a Anulacion
		var fechaRef, fechaEmi time.Time
		if err := rows.Scan(&a.ID, &a.Serie, &a.Correlativo, &fechaRef, &fechaEmi,
			&a.Estado, &a.Ticket, &a.SunatCodigo, &a.SunatMensaje, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.FechaReferencia = fechaRef.Format("2006-01-02")
		a.FechaEmision = fechaEmi.Format("2006-01-02")
		out = append(out, a)
	}
	return out, nil
}

// Items devuelve los comprobantes incluidos en una anulación con su motivo.
func (s *Store) Items(ctx context.Context, tenantID, id uuid.UUID) ([]ComprobanteAnular, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.tipo_documento, c.serie, c.correlativo, c.fecha_emision,
		       c.receptor_razon, c.total, c.moneda, ai.motivo
		FROM anulacion_items ai
		JOIN comprobantes c ON c.id = ai.comprobante_id
		WHERE ai.anulacion_id = $1 AND c.tenant_id = $2
		ORDER BY c.correlativo ASC
	`, id, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ComprobanteAnular
	for rows.Next() {
		var c ComprobanteAnular
		if err := rows.Scan(&c.ID, &c.Tipo, &c.Serie, &c.Correlativo, &c.FechaEmision,
			&c.ReceptorRazon, &c.Total, &c.Moneda, &c.Motivo); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// ItemIDs solo los UUIDs (para marcar comprobantes como anulados al aceptarse).
func (s *Store) ItemIDs(ctx context.Context, anulacionID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT comprobante_id FROM anulacion_items WHERE anulacion_id=$1
	`, anulacionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func (s *Store) MarcarEnviando(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE anulaciones SET estado='enviando', updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND estado IN ('pendiente','error')
	`, tenantID, id)
	return err
}

func (s *Store) MarcarConsultando(ctx context.Context, tenantID, id uuid.UUID, ticket string, xmlB64 string) error {
	var xmlBytes []byte
	if xmlB64 != "" {
		if b, err := base64.StdEncoding.DecodeString(xmlB64); err == nil {
			xmlBytes = b
		}
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE anulaciones SET estado='consultando', ticket=$3,
			xml_enviado=COALESCE($4, xml_enviado), updated_at=now()
		WHERE tenant_id=$1 AND id=$2
	`, tenantID, id, ticket, xmlBytes)
	return err
}

func (s *Store) AplicarResultado(ctx context.Context, tenantID, id uuid.UUID, estado, codigo, mensaje, cdrB64 string) error {
	var cdrBytes []byte
	if cdrB64 != "" {
		if b, err := base64.StdEncoding.DecodeString(cdrB64); err == nil {
			cdrBytes = b
		}
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE anulaciones SET estado=$3,
			sunat_codigo=NULLIF($4,''), sunat_mensaje=NULLIF($5,''),
			cdr_zip=COALESCE($6, cdr_zip), updated_at=now()
		WHERE tenant_id=$1 AND id=$2
	`, tenantID, id, estado, codigo, mensaje, cdrBytes)
	return err
}

func (s *Store) MarcarError(ctx context.Context, tenantID, id uuid.UUID, mensaje string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE anulaciones SET estado='error', sunat_mensaje=$3, updated_at=now()
		WHERE tenant_id=$1 AND id=$2
	`, tenantID, id, mensaje)
	return err
}
