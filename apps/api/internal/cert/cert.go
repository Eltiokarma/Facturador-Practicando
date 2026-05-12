// Package cert carga un .p12 protegido por passphrase y lo mantiene en
// memoria como PEM (cert + private key). La passphrase nunca se persiste.
package cert

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"software.sslmate.com/src/go-pkcs12"
)

type Material struct {
	CertPEM string // -----BEGIN CERTIFICATE----- ...
	KeyPEM  string // -----BEGIN PRIVATE KEY----- ...  (PKCS#8)
}

// Load lee el archivo .p12 en path y lo descifra con passphrase.
// Devuelve PEM listo para mandar al motor.
func Load(path, passphrase string) (*Material, error) {
	if path == "" {
		return nil, errors.New("cert: path vacío")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cert: leer %s: %w", path, err)
	}
	priv, leaf, _, err := pkcs12.DecodeChain(raw, passphrase)
	if err != nil {
		return nil, fmt.Errorf("cert: descifrar .p12 (¿passphrase correcta?): %w", err)
	}
	rsaKey, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("cert: la llave privada no es RSA (SUNAT requiere RSA)")
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	if err != nil {
		return nil, fmt.Errorf("cert: serializar key PKCS8: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leaf.Raw})
	return &Material{CertPEM: string(certPEM), KeyPEM: string(keyPEM)}, nil
}
