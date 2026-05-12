// Command seed-user crea el primer usuario del tenant configurado por env.
//
// Uso típico desde el host:
//   docker compose exec api /app/seed-user -email=tu@email -password=loquesea -nombre="Gerson" -rol=dueno
//
// O en local:
//   POSTGRES_HOST=localhost ... go run ./cmd/seed-user -email=... -password=...
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eltiokarma/facturador/apps/api/internal/config"
	"github.com/eltiokarma/facturador/apps/api/internal/db"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
	"github.com/eltiokarma/facturador/apps/api/internal/users"
)

func main() {
	var (
		email    = flag.String("email", "", "email del usuario (obligatorio)")
		password = flag.String("password", "", "password del usuario (obligatorio, min 8 chars)")
		nombre   = flag.String("nombre", "", "nombre del usuario (obligatorio)")
		rol      = flag.String("rol", "dueno", "rol: dueno | contador | cajero")
	)
	flag.Parse()
	if *email == "" || *password == "" || *nombre == "" {
		fmt.Fprintln(os.Stderr, "uso: seed-user -email=... -password=... -nombre=... [-rol=dueno]")
		os.Exit(2)
	}
	if len(*password) < 8 {
		fmt.Fprintln(os.Stderr, "password debe tener al menos 8 caracteres")
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		die("config", err)
	}
	ten, err := tenant.FromEnv()
	if err != nil {
		die("tenant", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.PostgresDSN)
	if err != nil {
		die("conectar DB", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		die("migrate", err)
	}

	tenantID, err := ensureTenant(ctx, pool, ten)
	if err != nil {
		die("ensureTenant", err)
	}

	store := users.NewStore(pool)
	id, err := store.Create(ctx, tenantID, *email, *nombre, *password, users.Rol(*rol))
	if err != nil {
		die("crear usuario", err)
	}

	fmt.Printf("✓ usuario creado\n  id:     %s\n  email:  %s\n  rol:    %s\n  tenant: %s (RUC %s)\n",
		id, *email, *rol, tenantID, ten.RUC)
}

func die(msg string, err error) {
	fmt.Fprintf(os.Stderr, "✗ %s: %v\n", msg, err)
	os.Exit(1)
}

func ensureTenant(ctx context.Context, pool *pgxpool.Pool, t *tenant.Tenant) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO tenants (ruc, razon_social, nombre_comercial, direccion_fiscal, ubigeo, sunat_mode)
		VALUES ($1,$2,$3,$4,$5,'beta')
		ON CONFLICT (ruc) DO UPDATE SET
			razon_social = EXCLUDED.razon_social,
			updated_at = now()
		RETURNING id
	`, t.RUC, t.RazonSocial, t.NombreComercial, t.DireccionFiscal, t.Ubigeo).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, errors.New("no se pudo crear tenant")
	}
	return id, err
}
