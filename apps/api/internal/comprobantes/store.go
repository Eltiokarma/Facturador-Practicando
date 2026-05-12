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
// Si la serie no existe, la crea con correlativo=1.
func (s *Store) NextCorrelativo(ctx context.Context, tenantID uuid.UUID, tipoDoc, serie string) (int64, error) {
	var next int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO series (tenant_id, tipo_documento, serie, correlativo)
		VALUES ($1,$2,$3,1)
		ON CONFLICT (tenant_id, tipo_documento, serie)
		DO UPDATE SET correlativo = series.correlativo + 1
		RETURNING correlativo
	`, tenantID, tipoDoc, serie).Scan(&next)
	if err != nil {
		return 0, err
	}
	return next, nil
}

type SaveInput struct {
	TenantID         uuid.UUID
	Tipo             string
	Serie            string
	Correlativo      int64
	FechaEmision     string
	Moneda           string
	ReceptorTipoDoc  string
	ReceptorNumDoc   string
	ReceptorRazon    string
	Gravado, Exonerado, Inafecto, Gratuito, IGV, ISC, ICBPER, Total float64
	Estado           string
	SunatCodigo      string
	SunatMensaje     string
	HashCPE          string
	XMLFirmadoB64    string
	CDRZipB64        string
	Payload          json.RawMessage
}

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

func (s *Store) Save(ctx context.Context, in SaveInput) (uuid.UUID, error) {
	id := uuid.New()
	dir := filepath.Join(s.dataDir, fmt.Sprintf("%s-%s-%08d", in.Tipo, in.Serie, in.Correlativo))
	_ = os.MkdirAll(dir, 0o750)

	var xmlBytes, cdrBytes []byte
	if in.XMLFirmadoB64 != "" {
		if b, err := base64.StdEncoding.DecodeString(in.XMLFirmadoB64); err == nil {
			xmlBytes = b
			_ = os.WriteFile(filepath.Join(dir, "firmado.xml"), b, 0o640)
		}
	}
	if in.CDRZipB64 != "" {
		if b, err := base64.StdEncoding.DecodeString(in.CDRZipB64); err == nil {
			cdrBytes = b
			_ = os.WriteFile(filepath.Join(dir, "cdr.zip"), b, 0o640)
		}
	}

	if len(in.Payload) == 0 {
		in.Payload = []byte("{}")
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO comprobantes (
			id, tenant_id, tipo_documento, serie, correlativo, fecha_emision, moneda,
			receptor_tipo_doc, receptor_num_doc, receptor_razon,
			total_gravado, total_exonerado, total_inafecto, total_gratuito,
			igv, isc, icbper, total,
			estado, sunat_codigo, sunat_mensaje,
			xml_enviado, cdr_zip, hash_cpe, payload
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,
			$8,$9,$10,
			$11,$12,$13,$14,
			$15,$16,$17,$18,
			$19,$20,$21,
			$22,$23,$24,$25
		)
	`,
		id, in.TenantID, in.Tipo, in.Serie, in.Correlativo, in.FechaEmision, in.Moneda,
		in.ReceptorTipoDoc, in.ReceptorNumDoc, in.ReceptorRazon,
		in.Gravado, in.Exonerado, in.Inafecto, in.Gratuito,
		in.IGV, in.ISC, in.ICBPER, in.Total,
		in.Estado, nullable(in.SunatCodigo), nullable(in.SunatMensaje),
		xmlBytes, cdrBytes, nullable(in.HashCPE), []byte(in.Payload),
	)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type ListFilter struct {
	TenantID uuid.UUID
	Estado   string // opcional
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
	query += ` ORDER BY created_at DESC LIMIT ` + intToStr(f.Limit) + ` OFFSET ` + intToStr(f.Offset)

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

// XML devuelve el XML firmado del comprobante.
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

// CDR devuelve el ZIP de la Constancia de Recepción.
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

func intToStr(n int) string {
	return fmt.Sprintf("%d", n)
}
