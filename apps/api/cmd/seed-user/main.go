// Command seed-user crea el primer usuario y, si la DB está vacía y hay
// TENANT_* en env, también crea el tenant inicial.
//
// Uso típico desde el host:
//   docker compose exec api /app/seed-user \
//       -email=tu@email -password=loquesea -nombre="Gerson" -rol=dueno
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/eltiokarma/facturador/apps/api/internal/config"
	mycrypto "github.com/eltiokarma/facturador/apps/api/internal/crypto"
	"github.com/eltiokarma/facturador/apps/api/internal/db"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
	"github.com/eltiokarma/facturador/apps/api/internal/users"
)

func main() {
	var (
		email    = flag.String("email", "", "email del usuario (obligatorio)")
		password = flag.String("password", "", "password (obligatorio, min 8 chars)")
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

	cipher, err := mycrypto.New(cfg.MasterKeyHex)
	if err != nil {
		die("crypto", err)
	}
	store := tenant.NewStore(pool, cipher)

	// Tomar el primer tenant disponible. Si no hay, crear desde env.
	tenants, err := store.ListAll(ctx)
	if err != nil {
		die("listar tenants", err)
	}
	if len(tenants) == 0 {
		in := tenant.FromEnv()
		if in == nil {
			die("no hay tenants", fmt.Errorf("definí TENANT_RUC y TENANT_RAZON_SOCIAL en el .env, o creá el tenant desde la UI primero"))
		}
		id, err := store.Create(ctx, tenant.CreateInput{
			RUC: in.RUC, RazonSocial: in.RazonSocial,
			NombreComercial: in.NombreComercial,
			DireccionFiscal: in.DireccionFiscal, Ubigeo: in.Ubigeo,
		})
		if err != nil {
			die("crear tenant", err)
		}
		if in.UsuarioSOL != "" && in.ClaveSOL != "" {
			_ = store.UpdateCredencialesSOL(ctx, id, in.UsuarioSOL, in.ClaveSOL)
		}
		tenants, _ = store.ListAll(ctx)
	}

	if len(tenants) == 0 {
		die("no hay tenants", fmt.Errorf("no se pudo crear el tenant inicial"))
	}
	tenantID := tenants[0].ID

	usersStore := users.NewStore(pool)
	id, err := usersStore.Create(ctx, tenantID, *email, *nombre, *password, users.Rol(*rol))
	if err != nil {
		die("crear usuario", err)
	}
	// Membresía en user_tenants
	if err := store.AddMember(ctx, tenantID, id, *rol); err != nil {
		die("add_member", err)
	}

	fmt.Printf("✓ usuario creado\n  id:     %s\n  email:  %s\n  rol:    %s\n  tenant: %s (RUC %s)\n",
		id, *email, *rol, tenantID, tenants[0].RUC)
}

func die(msg string, err error) {
	fmt.Fprintf(os.Stderr, "✗ %s: %v\n", msg, err)
	os.Exit(1)
}
