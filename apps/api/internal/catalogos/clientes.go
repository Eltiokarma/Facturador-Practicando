// Package catalogos persiste y consulta clientes y productos por tenant.
package catalogos

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("catalogos: no encontrado")

type Cliente struct {
	ID          uuid.UUID `json:"id"`
	TipoDoc     string    `json:"tipo_doc"`
	NumDoc      string    `json:"num_doc"`
	RazonSocial string    `json:"razon_social"`
	Direccion   string    `json:"direccion,omitempty"`
	Email       string    `json:"email,omitempty"`
	Telefono    string    `json:"telefono,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type ClientesStore struct {
	pool *pgxpool.Pool
}

func NewClientesStore(p *pgxpool.Pool) *ClientesStore { return &ClientesStore{pool: p} }

// Upsert crea o actualiza un cliente por (tenant, tipo_doc, num_doc).
func (s *ClientesStore) Upsert(ctx context.Context, tenantID uuid.UUID, c *Cliente) error {
	c.NumDoc = strings.TrimSpace(c.NumDoc)
	c.RazonSocial = strings.TrimSpace(c.RazonSocial)
	if c.NumDoc == "" || c.RazonSocial == "" {
		return errors.New("num_doc y razon_social son requeridos")
	}
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO clientes (id, tenant_id, tipo_doc, num_doc, razon_social, direccion, email, telefono)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (tenant_id, tipo_doc, num_doc) DO UPDATE SET
			razon_social = EXCLUDED.razon_social,
			direccion    = EXCLUDED.direccion,
			email        = EXCLUDED.email,
			telefono     = EXCLUDED.telefono,
			activo       = TRUE,
			updated_at   = now()
	`, c.ID, tenantID, c.TipoDoc, c.NumDoc, c.RazonSocial, nullIfEmpty(c.Direccion),
		nullIfEmpty(c.Email), nullIfEmpty(c.Telefono))
	return err
}

func (s *ClientesStore) List(ctx context.Context, tenantID uuid.UUID, q string, limit int) ([]Cliente, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q = strings.TrimSpace(q)
	var rows pgx.Rows
	var err error
	if q == "" {
		rows, err = s.pool.Query(ctx, `
			SELECT id, tipo_doc, num_doc, razon_social, COALESCE(direccion,''), COALESCE(email,''), COALESCE(telefono,''), created_at
			FROM clientes WHERE tenant_id=$1 AND activo
			ORDER BY razon_social ASC LIMIT $2
		`, tenantID, limit)
	} else {
		pat := "%" + strings.ToLower(q) + "%"
		rows, err = s.pool.Query(ctx, `
			SELECT id, tipo_doc, num_doc, razon_social, COALESCE(direccion,''), COALESCE(email,''), COALESCE(telefono,''), created_at
			FROM clientes
			WHERE tenant_id=$1 AND activo
			  AND (lower(razon_social) LIKE $2 OR num_doc LIKE $3)
			ORDER BY razon_social ASC LIMIT $4
		`, tenantID, pat, q+"%", limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Cliente
	for rows.Next() {
		var c Cliente
		if err := rows.Scan(&c.ID, &c.TipoDoc, &c.NumDoc, &c.RazonSocial,
			&c.Direccion, &c.Email, &c.Telefono, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *ClientesStore) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE clientes SET activo=FALSE, updated_at=now()
		WHERE tenant_id=$1 AND id=$2
	`, tenantID, id)
	return err
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
