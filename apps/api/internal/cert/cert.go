// Package cert maneja la carga y caché en memoria de los certificados
// digitales .p12 por tenant.
//
// Cada tenant tiene su propio .p12. El archivo se guarda en
//   <root>/<tenant_id>.p12
// y su passphrase, cifrada con AES-256-GCM (paquete crypto), se persiste
// en tenants.cert_pass_cifrado.
//
// Esto permite reinicios automáticos del container sin intervención humana.
// Trade-off conocido: si la MASTER_KEY se compromete junto con la DB y los
// .p12, se puede firmar comprobantes. Se asume que la MASTER_KEY se trata
// con el mismo cuidado que la passphrase del .p12 original. Si el operador
// quiere "cero persistencia" puede no usar este flow y seguir pasando
// CERT_PASSPHRASE por env.
package cert

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"software.sslmate.com/src/go-pkcs12"

	mycrypto "github.com/eltiokarma/facturador/apps/api/internal/crypto"
)

type Material struct {
	CertPEM string
	KeyPEM  string
}

// Manager mantiene un caché en memoria { tenantID → Material }.
// Es seguro para concurrencia.
type Manager struct {
	mu      sync.RWMutex
	items   map[uuid.UUID]*Material
	rootDir string
	cipher  *mycrypto.Cipher
}

func NewManager(rootDir string, c *mycrypto.Cipher) (*Manager, error) {
	if err := os.MkdirAll(rootDir, 0o750); err != nil {
		return nil, err
	}
	return &Manager{
		items:   make(map[uuid.UUID]*Material),
		rootDir: rootDir,
		cipher:  c,
	}, nil
}

// Get devuelve el material descifrado para un tenant. Si no está cargado,
// devuelve ErrNoCargado.
func (m *Manager) Get(tenantID uuid.UUID) (*Material, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mat, ok := m.items[tenantID]
	if !ok {
		return nil, ErrNoCargado
	}
	return mat, nil
}

var ErrNoCargado = errors.New("cert: no hay certificado cargado para este tenant")

// PathFor devuelve la ruta donde vive (o debería vivir) el .p12 de un tenant.
func (m *Manager) PathFor(tenantID uuid.UUID) string {
	return filepath.Join(m.rootDir, tenantID.String()+".p12")
}

// SaveAndLoad guarda el .p12 a disco, intenta descifrarlo con la passphrase,
// y si funciona lo cachea. Devuelve también la passphrase cifrada para
// que el caller la guarde en DB.
func (m *Manager) SaveAndLoad(tenantID uuid.UUID, p12Bytes []byte, passphrase string) (passCipher []byte, err error) {
	path := m.PathFor(tenantID)
	if err := os.WriteFile(path, p12Bytes, 0o600); err != nil {
		return nil, fmt.Errorf("cert: guardar %s: %w", path, err)
	}
	mat, err := loadP12(path, passphrase)
	if err != nil {
		// Si la descarga falló, borramos el archivo para no dejar basura.
		_ = os.Remove(path)
		return nil, err
	}
	m.mu.Lock()
	m.items[tenantID] = mat
	m.mu.Unlock()

	blob, err := m.cipher.Encrypt(passphrase)
	if err != nil {
		return nil, err
	}
	return blob, nil
}

// LoadFromDisk lee el .p12 del disco y la passphrase cifrada de DB.
// Se llama al arrancar el proceso, por cada tenant existente.
func (m *Manager) LoadFromDisk(tenantID uuid.UUID, certPath string, passBlob []byte) error {
	if certPath == "" {
		certPath = m.PathFor(tenantID)
	}
	if _, err := os.Stat(certPath); err != nil {
		return fmt.Errorf("cert: archivo %s no existe: %w", certPath, err)
	}
	pass, err := m.cipher.Decrypt(passBlob)
	if err != nil {
		return fmt.Errorf("cert: descifrar passphrase: %w", err)
	}
	if pass == "" {
		return errors.New("cert: passphrase no persistida para este tenant")
	}
	mat, err := loadP12(certPath, pass)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.items[tenantID] = mat
	m.mu.Unlock()
	return nil
}

// HasCert devuelve true si hay un certificado cargado en memoria.
func (m *Manager) HasCert(tenantID uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.items[tenantID]
	return ok
}

// loadP12 abre el archivo, descifra y devuelve PEM cert + PEM key.
func loadP12(path, passphrase string) (*Material, error) {
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
