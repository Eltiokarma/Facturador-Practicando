// Package tenant: en modo multi-tenant, los datos vivos del tenant viven
// en Record (DB-backed, cacheados por Manager). El struct BootstrapInput
// solo se usa al arrancar el proceso para crear el tenant inicial si la
// DB está vacía. Los env vars TENANT_* son opt-in y solo aplican al
// primer arranque.
package tenant

import (
	"os"
	"regexp"
)

type BootstrapInput struct {
	RUC             string
	RazonSocial     string
	NombreComercial string
	DireccionFiscal string
	Ubigeo          string
	Departamento    string
	Provincia       string
	Distrito        string
	UsuarioSOL      string
	ClaveSOL        string
	CertPath        string
	CertPassphrase  string
}

var bootstrapRUCRe = regexp.MustCompile(`^\d{11}$`)

// FromEnv lee las TENANT_* del proceso. Devuelve nil si no hay TENANT_RUC,
// indicando "no auto-bootstrappear". Esto permite arrancar el sistema sin
// estado preexistente y crear el primer tenant desde la UI.
func FromEnv() *BootstrapInput {
	ruc := os.Getenv("TENANT_RUC")
	if ruc == "" {
		return nil
	}
	if !bootstrapRUCRe.MatchString(ruc) {
		return nil
	}
	return &BootstrapInput{
		RUC:             ruc,
		RazonSocial:     os.Getenv("TENANT_RAZON_SOCIAL"),
		NombreComercial: os.Getenv("TENANT_NOMBRE_COMERCIAL"),
		DireccionFiscal: getEnv("TENANT_DIRECCION_FISCAL", "-"),
		Ubigeo:          getEnv("TENANT_UBIGEO", "150101"),
		Departamento:    getEnv("TENANT_DEPARTAMENTO", "LIMA"),
		Provincia:       getEnv("TENANT_PROVINCIA", "LIMA"),
		Distrito:        getEnv("TENANT_DISTRITO", "LIMA"),
		UsuarioSOL:      os.Getenv("TENANT_USUARIO_SOL"),
		ClaveSOL:        os.Getenv("TENANT_CLAVE_SOL"),
		CertPath:        getEnv("CERT_PATH", "/app/certs/cert.p12"),
		CertPassphrase:  os.Getenv("CERT_PASSPHRASE"),
	}
}

func getEnv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}
