package tenant

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	mycrypto "github.com/eltiokarma/facturador/apps/api/internal/crypto"
)

// Record es lo que vive en la tabla tenants, con las credenciales SOL
// descifradas en memoria.
type Record struct {
	ID              uuid.UUID `json:"id"`
	RUC             string    `json:"ruc"`
	RazonSocial     string    `json:"razon_social"`
	NombreComercial string    `json:"nombre_comercial,omitempty"`
	DireccionFiscal string    `json:"direccion_fiscal,omitempty"`
	Ubigeo          string    `json:"ubigeo,omitempty"`
	Departamento    string    `json:"departamento,omitempty"`
	Provincia       string    `json:"provincia,omitempty"`
	Distrito        string    `json:"distrito,omitempty"`
	SunatMode       string    `json:"sunat_mode"`
	UsuarioSOL      string    `json:"-"` // sensible, no se serializa
	ClaveSOL        string    `json:"-"`
	CertPath        string    `json:"cert_path,omitempty"`
	CertPassCipher  []byte    `json:"-"`
	DemoMode        bool      `json:"demo_mode"`
	// Credenciales API GRE (OAuth2). Solo en memoria.
	GREClientID     string    `json:"-"`
	GREClientSecret string    `json:"-"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Store struct {
	pool *pgxpool.Pool
	c    *mycrypto.Cipher
}

func NewStore(pool *pgxpool.Pool, c *mycrypto.Cipher) *Store {
	return &Store{pool: pool, c: c}
}

var (
	ErrNotFound = errors.New("tenant: no encontrado")
	rucRe       = regexp.MustCompile(`^\d{11}$`)
)

func (s *Store) ByID(ctx context.Context, id uuid.UUID) (*Record, error) {
	var r Record
	var usuario, clave, passp, greID, greSecret []byte
	var ubigeo, certPath, nombre, direccion *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, ruc, razon_social, nombre_comercial, direccion_fiscal,
		       ubigeo, sunat_mode,
		       sunat_usuario_sol_cifrado, sunat_clave_sol_cifrada,
		       cert_path, cert_pass_cifrado, demo_mode,
		       gre_client_id_cifrado, gre_client_secret_cifrado,
		       updated_at
		FROM tenants WHERE id=$1
	`, id).Scan(&r.ID, &r.RUC, &r.RazonSocial, &nombre, &direccion,
		&ubigeo, &r.SunatMode, &usuario, &clave, &certPath, &passp, &r.DemoMode,
		&greID, &greSecret, &r.UpdatedAt)
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
	r.CertPassCipher = passp
	r.UsuarioSOL, err = s.c.Decrypt(usuario)
	if err != nil {
		return nil, err
	}
	r.ClaveSOL, err = s.c.Decrypt(clave)
	if err != nil {
		return nil, err
	}
	r.GREClientID, err = s.c.Decrypt(greID)
	if err != nil {
		return nil, err
	}
	r.GREClientSecret, err = s.c.Decrypt(greSecret)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdateGRECredenciales cifra y guarda las credenciales OAuth2 de la API GRE.
func (s *Store) UpdateGRECredenciales(ctx context.Context, id uuid.UUID, clientID, clientSecret string) error {
	idBlob, err := s.c.Encrypt(clientID)
	if err != nil {
		return err
	}
	secretBlob, err := s.c.Encrypt(clientSecret)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE tenants SET
			gre_client_id_cifrado     = $2,
			gre_client_secret_cifrado = $3,
			updated_at                = now()
		WHERE id=$1
	`, id, idBlob, secretBlob)
	return err
}

func (s *Store) ByRUC(ctx context.Context, ruc string) (*Record, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM tenants WHERE ruc=$1`, ruc).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s *Store) ListAll(ctx context.Context) ([]*Record, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM tenants ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	out := make([]*Record, 0, len(ids))
	for _, id := range ids {
		rec, err := s.ByID(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

// ByUser devuelve los tenants a los que el usuario tiene acceso.
func (s *Store) ByUser(ctx context.Context, userID uuid.UUID) ([]*Record, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT tenant_id FROM user_tenants WHERE user_id=$1
	`, userID)
	if err != nil {
		return nil, err
	}
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	out := make([]*Record, 0, len(ids))
	for _, id := range ids {
		rec, err := s.ByID(ctx, id)
		if err != nil {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

type CreateInput struct {
	RUC             string
	RazonSocial     string
	NombreComercial string
	DireccionFiscal string
	Ubigeo          string
}

func (s *Store) Create(ctx context.Context, in CreateInput) (uuid.UUID, error) {
	in.RUC = strings.TrimSpace(in.RUC)
	in.RazonSocial = strings.TrimSpace(in.RazonSocial)
	if !rucRe.MatchString(in.RUC) {
		return uuid.Nil, errors.New("RUC inválido")
	}
	if in.RazonSocial == "" {
		return uuid.Nil, errors.New("razón social requerida")
	}
	// Por defecto demo_mode=true: el dueño activa producción real desde
	// Configuración una vez que tiene cert + credenciales.
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO tenants (ruc, razon_social, nombre_comercial, direccion_fiscal, ubigeo, sunat_mode, demo_mode)
		VALUES ($1,$2,$3,$4,$5,'beta', TRUE)
		ON CONFLICT (ruc) DO NOTHING
		RETURNING id
	`, in.RUC, in.RazonSocial, nullable(in.NombreComercial),
		nullable(in.DireccionFiscal), nullable(in.Ubigeo)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, errors.New("ya existe un tenant con ese RUC")
	}
	return id, err
}

// SetDemoMode permite al dueño desactivar el modo demo. Requiere haber
// cargado cert y credenciales antes.
func (s *Store) SetDemoMode(ctx context.Context, id uuid.UUID, demo bool) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenants SET demo_mode=$2, updated_at=now() WHERE id=$1
	`, id, demo)
	return err
}

// AddMember agrega un usuario a un tenant con cierto rol.
func (s *Store) AddMember(ctx context.Context, tenantID, userID uuid.UUID, rol string) error {
	if rol != "dueno" && rol != "contador" && rol != "cajero" {
		return errors.New("rol inválido")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_tenants (user_id, tenant_id, rol)
		VALUES ($1,$2,$3)
		ON CONFLICT (user_id, tenant_id) DO UPDATE SET rol = EXCLUDED.rol
	`, userID, tenantID, rol)
	return err
}

// IsMember chequea si user tiene acceso a tenant.
func (s *Store) IsMember(ctx context.Context, userID, tenantID uuid.UUID) (bool, string, error) {
	var rol string
	err := s.pool.QueryRow(ctx, `
		SELECT rol FROM user_tenants WHERE user_id=$1 AND tenant_id=$2
	`, userID, tenantID).Scan(&rol)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, rol, nil
}

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
		return errors.New("razón social requerida")
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

// BootstrapCredenciales copia las credenciales del env a DB la primera vez.
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

// SetCertRef registra el path y la passphrase cifrada del .p12 del tenant.
func (s *Store) SetCertRef(ctx context.Context, id uuid.UUID, certPath string, passCipher []byte) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenants SET cert_path=$2, cert_pass_cifrado=$3, updated_at=now()
		WHERE id=$1
	`, id, certPath, passCipher)
	return err
}

func nullable(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
