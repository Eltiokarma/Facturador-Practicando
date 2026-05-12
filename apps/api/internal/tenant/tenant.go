// Package tenant resuelve la configuración del tenant. Para el MVP "hola mundo"
// solo se soporta un tenant configurado por env vars. La versión multi-tenant
// completa (DB + auth) llega después.
package tenant

import (
	"errors"
	"fmt"
	"os"
	"regexp"
)

type Tenant struct {
	RUC              string
	RazonSocial      string
	NombreComercial  string
	DireccionFiscal  string
	Ubigeo           string
	Departamento     string
	Provincia        string
	Distrito         string
	UsuarioSOL       string
	ClaveSOL         string
	CertPath         string
	CertPassphrase   string
}

var rucRe = regexp.MustCompile(`^\d{11}$`)

// FromEnv lee la config del único tenant desde variables de entorno.
func FromEnv() (*Tenant, error) {
	t := &Tenant{
		RUC:             os.Getenv("TENANT_RUC"),
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
	if t.RUC == "" {
		return nil, errors.New("tenant: TENANT_RUC no definido")
	}
	if !rucRe.MatchString(t.RUC) {
		return nil, fmt.Errorf("tenant: TENANT_RUC inválido (%q), debe ser 11 dígitos", t.RUC)
	}
	if t.RazonSocial == "" {
		return nil, errors.New("tenant: TENANT_RAZON_SOCIAL no definido")
	}
	if t.UsuarioSOL == "" || t.ClaveSOL == "" {
		return nil, errors.New("tenant: TENANT_USUARIO_SOL y TENANT_CLAVE_SOL son obligatorios (usar el usuario SECUNDARIO, no el principal)")
	}
	if t.CertPassphrase == "" {
		return nil, errors.New("tenant: CERT_PASSPHRASE no definida (la passphrase del .p12 nunca se persiste, hay que pasarla al arrancar)")
	}
	if t.NombreComercial == "" {
		t.NombreComercial = t.RazonSocial
	}
	return t, nil
}

func getEnv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}
