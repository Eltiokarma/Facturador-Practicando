// Package storage persiste artefactos de emisión en filesystem.
// En el MVP "hola mundo" esto reemplaza temporalmente la DB. Después
// migra a Postgres, pero la capa que consume sigue siendo la misma.
package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Store struct {
	root string
}

func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

type Registro struct {
	Tipo         string          `json:"tipo"`
	Serie        string          `json:"serie"`
	Correlativo  int64           `json:"correlativo"`
	FechaEmision string          `json:"fecha_emision"`
	Estado       string          `json:"estado"`
	Codigo       string          `json:"codigo"`
	Mensaje      string          `json:"mensaje"`
	HashCPE      string          `json:"hash_cpe"`
	Payload      json.RawMessage `json:"payload"`
	GuardadoAt   time.Time       `json:"guardado_at"`
}

// Guardar persiste registro JSON, XML firmado y CDR en el filesystem.
// Estructura:
//   <root>/<tipo>-<serie>-<correlativo>/registro.json
//   <root>/<tipo>-<serie>-<correlativo>/firmado.xml
//   <root>/<tipo>-<serie>-<correlativo>/cdr.zip
func (s *Store) Guardar(reg Registro, xmlFirmadoB64, cdrZipB64 string) (string, error) {
	dir := filepath.Join(s.root, fmt.Sprintf("%s-%s-%08d", reg.Tipo, reg.Serie, reg.Correlativo))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	reg.GuardadoAt = time.Now().UTC()
	body, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "registro.json"), body, 0o640); err != nil {
		return "", err
	}
	if xmlFirmadoB64 != "" {
		xml, err := base64.StdEncoding.DecodeString(xmlFirmadoB64)
		if err == nil {
			_ = os.WriteFile(filepath.Join(dir, "firmado.xml"), xml, 0o640)
		}
	}
	if cdrZipB64 != "" {
		zip, err := base64.StdEncoding.DecodeString(cdrZipB64)
		if err == nil {
			_ = os.WriteFile(filepath.Join(dir, "cdr.zip"), zip, 0o640)
		}
	}
	return dir, nil
}
