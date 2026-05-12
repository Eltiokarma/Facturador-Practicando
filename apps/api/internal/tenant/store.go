package tenant

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	mycrypto "github.com/eltiokarma/facturador/apps/api/internal/crypto"
)

// Record es lo que vive en la tabla tenants (sin descifrar).
type Record struct {
	ID              uuid.UUID
	RUC             string
	RazonSocial     string
	NombreComercial string
	DireccionFiscal string
	Ubigeo          string
	Departamento    string
	Provincia       string
	Distrito        string
	SunatMode       string
	UsuarioSOL      string // descifrado en memoria
	ClaveSOL        string // descifrado en memoria
	CertPath        string
	UpdatedAt       time.Time
}

type Store struct {
	pool *pgxpool.Pool
	c    *mycrypto.Cipher
}

func NewStore(pool *pgxpool.Pool, c *mycrypto.Cipher) *Store {
	return &Store{pool: pool, c: c}
}

var ErrNotFound = errors.New("tenant: no encontrado")

// ByID lee el tenant con credenciales descifradas en memoria.
func (s *Store) ByID(ctx context.Context, id uuid.UUID) (*Record, error) {
	var r Record
	var usuario, clave []byte
	var ubigeo, certPath, nombre, direccion *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, ruc, razon_social, nombre_comercial, direccion_fiscal,
		       ubigeo, sunat_mode,
		       sunat_usuario_sol_cifrado, sunat_clave_sol_cifrada,
		       cert_path, updated_at
		FROM tenants WHERE id=$1
	`, id).Scan(&r.ID, &r.RUC, &r.RazonSocial, &nombre, &direccion,
		&ubigeo, &r.SunatMode, &usuario, &clave, &certPath, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if nombre != nil {
		r.NombreComercial = *nombre
	}
	if direccion != nil {
		r.DireccionFiscal = *direccion
	}
	if ubigeo != nil {
		r.Ubigeo = *ubigeo
	}
	if certPath != nil {
		r.CertPath = *certPath
	}
	r.UsuarioSOL, err = s.c.Decrypt(usuario)
	if err != nil {
		return nil, err
	}
	r.ClaveSOL, err = s.c.Decrypt(clave)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdatePerfil cambia datos no sensibles del tenant.
type PerfilInput struct {
	RazonSocial     string
	NombreComercial string
	DireccionFiscal string
	Ubigeo          string
	SunatMode       string
}

func (s *Store) UpdatePerfil(ctx context.Context, id uuid.UUID, in PerfilInput) error {
	if in.SunatMode != "beta" && in.SunatMode != "prod" {
		return errors.New("sunat_mode inválido")
	}
	if strings.TrimSpace(in.RazonSocial) == "" {
		return errors.New("razon_social requerida")
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE tenants SET
			razon_social     = $2,
			nombre_comercial = $3,
			direccion_fiscal = $4,
			ubigeo           = $5,
			sunat_mode       = $6,
			updated_at       = now()
		WHERE id=$1
	`, id, in.RazonSocial, nullable(in.NombreComercial),
		nullable(in.DireccionFiscal), nullable(in.Ubigeo), in.SunatMode)
	return err
}

// UpdateCredencialesSOL re-cifra y guarda usuario+clave SOL.
func (s *Store) UpdateCredencialesSOL(ctx context.Context, id uuid.UUID, usuario, clave string) error {
	uBlob, err := s.c.Encrypt(usuario)
	if err != nil {
		return err
	}
	kBlob, err := s.c.Encrypt(clave)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE tenants SET
			sunat_usuario_sol_cifrado = $2,
			sunat_clave_sol_cifrada   = $3,
			updated_at = now()
		WHERE id=$1
	`, id, uBlob, kBlob)
	return err
}

// Bootstrap: si las credenciales SOL aún no están en DB, las inserta
// usando el valor de env del proceso (que es el bootstrap inicial).
func (s *Store) BootstrapCredenciales(ctx context.Context, id uuid.UUID, usuario, clave string) error {
	var hasUser bool
	if err := s.pool.QueryRow(ctx, `
		SELECT (sunat_usuario_sol_cifrado IS NOT NULL) FROM tenants WHERE id=$1
	`, id).Scan(&hasUser); err != nil {
		return err
	}
	if hasUser {
		return nil
	}
	return s.UpdateCredencialesSOL(ctx, id, usuario, clave)
}

func nullable(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
