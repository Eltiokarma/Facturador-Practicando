// Package resumenes persiste resúmenes diarios (RC) de boletas.
package resumenes

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

var ErrNotFound = errors.New("resumenes: no encontrado")

type BoletaPendiente struct {
	ID            uuid.UUID `json:"id"`
	Serie         string    `json:"serie"`
	Correlativo   int64     `json:"correlativo"`
	ReceptorDoc   string    `json:"receptor_doc"`
	ReceptorRazon string    `json:"receptor_razon"`
	Moneda        string    `json:"moneda"`
	Gravado       float64   `json:"gravado"`
	Exonerado     float64   `json:"exonerado"`
	Inafecto      float64   `json:"inafecto"`
	IGV           float64   `json:"igv"`
	Total         float64   `json:"total"`
}

// PendientesPorFecha devuelve las boletas (tipo 03) aceptadas en SUNAT
// que aún NO están en ningún resumen aceptado, para la fecha dada.
func (s *Store) PendientesPorFecha(ctx context.Context, tenantID uuid.UUID, fecha time.Time) ([]BoletaPendiente, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.serie, c.correlativo,
		       c.receptor_num_doc, c.receptor_razon, c.moneda,
		       c.total_gravado, c.total_exonerado, c.total_inafecto, c.igv, c.total
		FROM comprobantes c
		WHERE c.tenant_id = $1
		  AND c.tipo_documento = '03'
		  AND c.fecha_emision = $2
		  AND c.estado IN ('aceptado','aceptado_con_obs')
		  AND NOT EXISTS (
		      SELECT 1 FROM resumen_items ri
		      JOIN resumenes_diarios r ON r.id = ri.resumen_id
		      WHERE ri.comprobante_id = c.id
		        AND r.estado IN ('pendiente','enviando','consultando','aceptado','aceptado_con_obs')
		  )
		ORDER BY c.correlativo ASC
	`, tenantID, fecha)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BoletaPendiente
	for rows.Next() {
		var b BoletaPendiente
		if err := rows.Scan(&b.ID, &b.Serie, &b.Correlativo,
			&b.ReceptorDoc, &b.ReceptorRazon, &b.Moneda,
			&b.Gravado, &b.Exonerado, &b.Inafecto, &b.IGV, &b.Total); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

type Resumen struct {
	ID                   uuid.UUID `json:"id"`
	Serie                string    `json:"serie"`
	Correlativo          int       `json:"correlativo"`
	FechaReferencia      string    `json:"fecha_referencia"`
	FechaEmision         string    `json:"fecha_emision"`
	Estado               string    `json:"estado"`
	Ticket               string    `json:"ticket,omitempty"`
	SunatCodigo          string    `json:"sunat_codigo,omitempty"`
	SunatMensaje         string    `json:"sunat_mensaje,omitempty"`
	CantidadComprobantes int       `json:"cantidad_comprobantes"`
	CreatedAt            time.Time `json:"created_at"`
}

// Crear inserta el resumen con estado=pendiente y vincula las boletas.
// El correlativo se asigna atómicamente: el siguiente N para esa serie.
func (s *Store) Crear(ctx context.Context, tenantID uuid.UUID, fechaRef time.Time, boletas []uuid.UUID) (*Resumen, error) {
	if len(boletas) == 0 {
		return nil, errors.New("no hay boletas para resumir")
	}
	serie := "RC-" + fechaRef.Format("20060102")

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var correlativo int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(correlativo), 0) + 1
		FROM resumenes_diarios WHERE tenant_id=$1 AND serie=$2
	`, tenantID, serie).Scan(&correlativo)
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	hoy := time.Now().UTC().Truncate(24 * time.Hour)
	_, err = tx.Exec(ctx, `
		INSERT INTO resumenes_diarios (
			id, tenant_id, serie, correlativo,
			fecha_referencia, fecha_emision,
			estado, cantidad_comprobantes
		) VALUES ($1,$2,$3,$4,$5,$6,'pendiente',$7)
	`, id, tenantID, serie, correlativo, fechaRef, hoy, len(boletas))
	if err != nil {
		return nil, err
	}

	for _, bid := range boletas {
		if _, err := tx.Exec(ctx, `
			INSERT INTO resumen_items (resumen_id, comprobante_id) VALUES ($1,$2)
		`, id, bid); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.Get(ctx, tenantID, id)
}

func (s *Store) Get(ctx context.Context, tenantID, id uuid.UUID) (*Resumen, error) {
	var r Resumen
	var fechaRef, fechaEmi time.Time
	var ticket, code, msg *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, serie, correlativo, fecha_referencia, fecha_emision,
		       estado, ticket, sunat_codigo, sunat_mensaje,
		       cantidad_comprobantes, created_at
		FROM resumenes_diarios WHERE tenant_id=$1 AND id=$2
	`, tenantID, id).Scan(&r.ID, &r.Serie, &r.Correlativo, &fechaRef, &fechaEmi,
		&r.Estado, &ticket, &code, &msg, &r.CantidadComprobantes, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	r.FechaReferencia = fechaRef.Format("2006-01-02")
	r.FechaEmision = fechaEmi.Format("2006-01-02")
	if ticket != nil {
		r.Ticket = *ticket
	}
	if code != nil {
		r.SunatCodigo = *code
	}
	if msg != nil {
		r.SunatMensaje = *msg
	}
	return &r, nil
}

func (s *Store) List(ctx context.Context, tenantID uuid.UUID, limit int) ([]Resumen, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, serie, correlativo, fecha_referencia, fecha_emision,
		       estado, COALESCE(ticket,''), COALESCE(sunat_codigo,''), COALESCE(sunat_mensaje,''),
		       cantidad_comprobantes, created_at
		FROM resumenes_diarios WHERE tenant_id=$1
		ORDER BY created_at DESC LIMIT %d
	`, limit), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Resumen
	for rows.Next() {
		var r Resumen
		var fechaRef, fechaEmi time.Time
		if err := rows.Scan(&r.ID, &r.Serie, &r.Correlativo, &fechaRef, &fechaEmi,
			&r.Estado, &r.Ticket, &r.SunatCodigo, &r.SunatMensaje,
			&r.CantidadComprobantes, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.FechaReferencia = fechaRef.Format("2006-01-02")
		r.FechaEmision = fechaEmi.Format("2006-01-02")
		out = append(out, r)
	}
	return out, nil
}

// Boletas devuelve las boletas vinculadas a un resumen.
func (s *Store) Boletas(ctx context.Context, tenantID, id uuid.UUID) ([]BoletaPendiente, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.serie, c.correlativo,
		       c.receptor_num_doc, c.receptor_razon, c.moneda,
		       c.total_gravado, c.total_exonerado, c.total_inafecto, c.igv, c.total
		FROM resumen_items ri
		JOIN comprobantes c ON c.id = ri.comprobante_id
		WHERE ri.resumen_id = $1 AND c.tenant_id = $2
		ORDER BY c.correlativo ASC
	`, id, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BoletaPendiente
	for rows.Next() {
		var b BoletaPendiente
		if err := rows.Scan(&b.ID, &b.Serie, &b.Correlativo,
			&b.ReceptorDoc, &b.ReceptorRazon, &b.Moneda,
			&b.Gravado, &b.Exonerado, &b.Inafecto, &b.IGV, &b.Total); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// MarcarEnviando transiciona el resumen al estado 'enviando'.
func (s *Store) MarcarEnviando(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE resumenes_diarios SET estado='enviando', updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND estado IN ('pendiente','error')
	`, tenantID, id)
	return err
}

// MarcarConsultando se llama después de que SUNAT devuelve un ticket.
func (s *Store) MarcarConsultando(ctx context.Context, tenantID, id uuid.UUID, ticket string, xmlB64 string) error {
	var xmlBytes []byte
	if xmlB64 != "" {
		if b, err := base64.StdEncoding.DecodeString(xmlB64); err == nil {
			xmlBytes = b
		}
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE resumenes_diarios SET
			estado='consultando', ticket=$3,
			xml_enviado=COALESCE($4, xml_enviado), updated_at=now()
		WHERE tenant_id=$1 AND id=$2
	`, tenantID, id, ticket, xmlBytes)
	return err
}

// AplicarResultado finaliza el resumen con el veredicto de SUNAT.
func (s *Store) AplicarResultado(ctx context.Context, tenantID, id uuid.UUID, estado, codigo, mensaje, cdrB64 string) error {
	var cdrBytes []byte
	if cdrB64 != "" {
		if b, err := base64.StdEncoding.DecodeString(cdrB64); err == nil {
			cdrBytes = b
		}
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE resumenes_diarios SET
			estado=$3, sunat_codigo=NULLIF($4,''), sunat_mensaje=NULLIF($5,''),
			cdr_zip=COALESCE($6, cdr_zip), updated_at=now()
		WHERE tenant_id=$1 AND id=$2
	`, tenantID, id, estado, codigo, mensaje, cdrBytes)
	return err
}

// MarcarError marca el resumen como error transitorio.
func (s *Store) MarcarError(ctx context.Context, tenantID, id uuid.UUID, mensaje string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE resumenes_diarios SET estado='error', sunat_mensaje=$3, updated_at=now()
		WHERE tenant_id=$1 AND id=$2
	`, tenantID, id, mensaje)
	return err
}
