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

type Producto struct {
	ID            uuid.UUID `json:"id"`
	Codigo        string    `json:"codigo,omitempty"`
	Descripcion   string    `json:"descripcion"`
	Unidad        string    `json:"unidad"`
	ValorUnitario float64   `json:"valor_unitario"`
	AfectacionIGV string    `json:"afectacion_igv"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProductosStore struct {
	pool *pgxpool.Pool
}

func NewProductosStore(p *pgxpool.Pool) *ProductosStore { return &ProductosStore{pool: p} }

func (s *ProductosStore) Upsert(ctx context.Context, tenantID uuid.UUID, p *Producto) error {
	p.Descripcion = strings.TrimSpace(p.Descripcion)
	if p.Descripcion == "" {
		return errors.New("descripcion requerida")
	}
	if p.Unidad == "" {
		p.Unidad = "NIU"
	}
	if p.AfectacionIGV == "" {
		p.AfectacionIGV = "10"
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
		_, err := s.pool.Exec(ctx, `
			INSERT INTO productos (id, tenant_id, codigo, descripcion, unidad, valor_unitario, afectacion_igv)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, p.ID, tenantID, nullIfEmpty(p.Codigo), p.Descripcion, p.Unidad, p.ValorUnitario, p.AfectacionIGV)
		return err
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE productos SET
			codigo = $1, descripcion = $2, unidad = $3,
			valor_unitario = $4, afectacion_igv = $5, activo = TRUE, updated_at = now()
		WHERE tenant_id = $6 AND id = $7
	`, nullIfEmpty(p.Codigo), p.Descripcion, p.Unidad, p.ValorUnitario, p.AfectacionIGV, tenantID, p.ID)
	return err
}

func (s *ProductosStore) List(ctx context.Context, tenantID uuid.UUID, q string, limit int) ([]Producto, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q = strings.TrimSpace(q)
	var rows pgx.Rows
	var err error
	if q == "" {
		rows, err = s.pool.Query(ctx, `
			SELECT id, COALESCE(codigo,''), descripcion, unidad, valor_unitario, afectacion_igv, created_at
			FROM productos WHERE tenant_id=$1 AND activo
			ORDER BY descripcion ASC LIMIT $2
		`, tenantID, limit)
	} else {
		pat := "%" + strings.ToLower(q) + "%"
		rows, err = s.pool.Query(ctx, `
			SELECT id, COALESCE(codigo,''), descripcion, unidad, valor_unitario, afectacion_igv, created_at
			FROM productos
			WHERE tenant_id=$1 AND activo
			  AND (lower(descripcion) LIKE $2 OR lower(COALESCE(codigo,'')) LIKE $2)
			ORDER BY descripcion ASC LIMIT $3
		`, tenantID, pat, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Producto
	for rows.Next() {
		var p Producto
		if err := rows.Scan(&p.ID, &p.Codigo, &p.Descripcion, &p.Unidad,
			&p.ValorUnitario, &p.AfectacionIGV, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *ProductosStore) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE productos SET activo=FALSE, updated_at=now()
		WHERE tenant_id=$1 AND id=$2
	`, tenantID, id)
	return err
}
