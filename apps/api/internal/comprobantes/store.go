// Package comprobantes persiste comprobantes en Postgres. Los artefactos
// binarios (XML firmado, CDR.zip) viven adicionalmente en filesystem
// para facilitar inspección por el operador.
package comprobantes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool    *pgxpool.Pool
	dataDir string
}

func NewStore(pool *pgxpool.Pool, dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, err
	}
	return &Store{pool: pool, dataDir: dataDir}, nil
}

// NextCorrelativo reserva el siguiente correlativo atómicamente.
func (s *Store) NextCorrelativo(ctx context.Context, tenantID uuid.UUID, tipoDoc, serie string) (int64, error) {
	var next int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO series (tenant_id, tipo_documento, serie, correlativo)
		VALUES ($1,$2,$3,1)
		ON CONFLICT (tenant_id, tipo_documento, serie)
		DO UPDATE SET correlativo = series.correlativo + 1
		RETURNING correlativo
	`, tenantID, tipoDoc, serie).Scan(&next)
	return next, err
}

// PendienteInput contiene los datos del comprobante listos para guardar
// con estado=pendiente, sin haber tocado SUNAT.
type PendienteInput struct {
	TenantID                                                        uuid.UUID
	Tipo, Serie                                                     string
	Correlativo                                                     int64
	FechaEmision, Moneda                                            string
	ReceptorTipoDoc, ReceptorNumDoc, ReceptorRazon                  string
	Gravado, Exonerado, Inafecto, Gratuito, IGV, ISC, ICBPER, Total float64
	Payload                                                         json.RawMessage
}

// CreatePendiente guarda un comprobante recién recibido con estado=pendiente.
func (s *Store) CreatePendiente(ctx context.Context, in PendienteInput) (uuid.UUID, error) {
	id := uuid.New()
	if len(in.Payload) == 0 {
		in.Payload = []byte("{}")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO comprobantes (
			id, tenant_id, tipo_documento, serie, correlativo, fecha_emision, moneda,
			receptor_tipo_doc, receptor_num_doc, receptor_razon,
			total_gravado, total_exonerado, total_inafecto, total_gratuito,
			igv, isc, icbper, total,
			estado, payload
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,
			$8,$9,$10,
			$11,$12,$13,$14,
			$15,$16,$17,$18,
			'pendiente',$19
		)
	`,
		id, in.TenantID, in.Tipo, in.Serie, in.Correlativo, in.FechaEmision, in.Moneda,
		in.ReceptorTipoDoc, in.ReceptorNumDoc, in.ReceptorRazon,
		in.Gravado, in.Exonerado, in.Inafecto, in.Gratuito,
		in.IGV, in.ISC, in.ICBPER, in.Total,
		[]byte(in.Payload),
	)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// MarcarEnviando pasa el comprobante de pendiente → enviando cuando el
// worker lo toma. Idempotente: si ya está en otro estado, no hace nada.
func (s *Store) MarcarEnviando(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE comprobantes SET estado='enviando', updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND estado IN ('pendiente','error','enviando')
	`, tenantID, id)
	return err
}

// MarcarError marca el comprobante como error transitorio (motor caído,
// timeout, etc.) — el worker puede reintentarlo.
func (s *Store) MarcarError(ctx context.Context, tenantID, id uuid.UUID, mensaje string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE comprobantes SET estado='error', sunat_mensaje=$3, updated_at=now()
		WHERE tenant_id=$1 AND id=$2
	`, tenantID, id, mensaje)
	return err
}

// Resultado contiene lo que el motor devuelve después de hablar con SUNAT.
type Resultado struct {
	Estado        string
	Codigo        string
	Mensaje       string
	HashCPE       string
	XMLFirmadoB64 string
	CDRZipB64     string
}

// AplicarResultado actualiza el comprobante con la respuesta de SUNAT.
// Guarda también XML y CDR como blobs en DB y como archivos en filesystem.
func (s *Store) AplicarResultado(ctx context.Context, tenantID, id uuid.UUID, r Resultado, tipo, serie string, correlativo int64) error {
	var xmlBytes, cdrBytes []byte
	if r.XMLFirmadoB64 != "" {
		if b, err := base64.StdEncoding.DecodeString(r.XMLFirmadoB64); err == nil {
			xmlBytes = b
		}
	}
	if r.CDRZipB64 != "" {
		if b, err := base64.StdEncoding.DecodeString(r.CDRZipB64); err == nil {
			cdrBytes = b
		}
	}

	// Best-effort: persistir también en filesystem
	if xmlBytes != nil || cdrBytes != nil {
		dir := filepath.Join(s.dataDir, fmt.Sprintf("%s-%s-%08d", tipo, serie, correlativo))
		if err := os.MkdirAll(dir, 0o750); err == nil {
			if xmlBytes != nil {
				_ = os.WriteFile(filepath.Join(dir, "firmado.xml"), xmlBytes, 0o640)
			}
			if cdrBytes != nil {
				_ = os.WriteFile(filepath.Join(dir, "cdr.zip"), cdrBytes, 0o640)
			}
		}
	}

	_, err := s.pool.Exec(ctx, `
		UPDATE comprobantes SET
			estado = $3,
			sunat_codigo = $4,
			sunat_mensaje = $5,
			hash_cpe = NULLIF($6,''),
			xml_enviado = COALESCE($7, xml_enviado),
			cdr_zip = COALESCE($8, cdr_zip),
			updated_at = now()
		WHERE tenant_id=$1 AND id=$2
	`, tenantID, id, r.Estado,
		nullable(r.Codigo), nullable(r.Mensaje), r.HashCPE,
		xmlBytes, cdrBytes,
	)
	return err
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ---- Lectura ---------------------------------------------------------

type Comprobante struct {
	ID              uuid.UUID `json:"id"`
	Tipo            string    `json:"tipo"`
	Serie           string    `json:"serie"`
	Correlativo     int64     `json:"correlativo"`
	FechaEmision    string    `json:"fecha_emision"`
	Moneda          string    `json:"moneda"`
	ReceptorTipoDoc string    `json:"receptor_tipo_doc"`
	ReceptorDoc     string    `json:"receptor_doc"`
	ReceptorRazon   string    `json:"receptor_razon"`
	Total           float64   `json:"total"`
	IGV             float64   `json:"igv"`
	Estado          string    `json:"estado"`
	SunatCodigo     string    `json:"sunat_codigo,omitempty"`
	SunatMensaje    string    `json:"sunat_mensaje,omitempty"`
	HashCPE         string    `json:"hash_cpe,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type Detalle struct {
	Comprobante
	Gravado   float64         `json:"gravado"`
	Exonerado float64         `json:"exonerado"`
	Inafecto  float64         `json:"inafecto"`
	Payload   json.RawMessage `json:"payload"`
}

type ListFilter struct {
	TenantID uuid.UUID
	Estado   string
	Limit    int
	Offset   int
}

func (s *Store) List(ctx context.Context, f ListFilter) ([]Comprobante, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	query := `
		SELECT id, tipo_documento, serie, correlativo, fecha_emision, moneda,
		       receptor_tipo_doc, receptor_num_doc, receptor_razon, total, igv, estado,
		       COALESCE(sunat_codigo,''), COALESCE(sunat_mensaje,''),
		       COALESCE(hash_cpe,''), created_at
		FROM comprobantes
		WHERE tenant_id = $1
	`
	args := []any{f.TenantID}
	if f.Estado != "" {
		query += ` AND estado = $2`
		args = append(args, f.Estado)
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d OFFSET %d`, f.Limit, f.Offset)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Comprobante
	for rows.Next() {
		var c Comprobante
		var fecha time.Time
		if err := rows.Scan(&c.ID, &c.Tipo, &c.Serie, &c.Correlativo, &fecha, &c.Moneda,
			&c.ReceptorTipoDoc, &c.ReceptorDoc, &c.ReceptorRazon, &c.Total, &c.IGV, &c.Estado,
			&c.SunatCodigo, &c.SunatMensaje, &c.HashCPE, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.FechaEmision = fecha.Format("2006-01-02")
		out = append(out, c)
	}
	return out, nil
}

var ErrNotFound = errors.New("comprobantes: no encontrado")

func (s *Store) Get(ctx context.Context, tenantID, id uuid.UUID) (*Detalle, error) {
	var d Detalle
	var fecha time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id, tipo_documento, serie, correlativo, fecha_emision, moneda,
		       receptor_tipo_doc, receptor_num_doc, receptor_razon, total, igv, estado,
		       COALESCE(sunat_codigo,''), COALESCE(sunat_mensaje,''),
		       COALESCE(hash_cpe,''), created_at,
		       total_gravado, total_exonerado, total_inafecto, payload
		FROM comprobantes WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(&d.ID, &d.Tipo, &d.Serie, &d.Correlativo, &fecha, &d.Moneda,
		&d.ReceptorTipoDoc, &d.ReceptorDoc, &d.ReceptorRazon, &d.Total, &d.IGV, &d.Estado,
		&d.SunatCodigo, &d.SunatMensaje, &d.HashCPE, &d.CreatedAt,
		&d.Gravado, &d.Exonerado, &d.Inafecto, &d.Payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d.FechaEmision = fecha.Format("2006-01-02")
	return &d, nil
}

func (s *Store) XML(ctx context.Context, tenantID, id uuid.UUID) ([]byte, error) {
	var data []byte
	err := s.pool.QueryRow(ctx, `
		SELECT xml_enviado FROM comprobantes WHERE tenant_id=$1 AND id=$2
	`, tenantID, id).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return data, err
}

func (s *Store) CDR(ctx context.Context, tenantID, id uuid.UUID) ([]byte, error) {
	var data []byte
	err := s.pool.QueryRow(ctx, `
		SELECT cdr_zip FROM comprobantes WHERE tenant_id=$1 AND id=$2
	`, tenantID, id).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return data, err
}
