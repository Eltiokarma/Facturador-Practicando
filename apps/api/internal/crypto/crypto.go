// Package crypto provee cifrado autenticado (AES-256-GCM) para datos
// sensibles que necesitamos persistir en DB: credenciales SOL, passphrase
// del cert, etc.
//
// La llave maestra viene de la env var MASTER_KEY. Debe ser exactamente
// 32 bytes en hex (64 caracteres). Se genera con:
//
//	openssl rand -hex 32
//
// Si la llave maestra cambia, todo lo cifrado anteriormente queda
// inaccesible. No es algo a rotar a la ligera.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

type Cipher struct {
	aead cipher.AEAD
}

// New construye un Cipher a partir de la llave maestra en hex.
func New(masterKeyHex string) (*Cipher, error) {
	if masterKeyHex == "" {
		return nil, errors.New("MASTER_KEY no definida")
	}
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, fmt.Errorf("MASTER_KEY no es hex: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("MASTER_KEY debe ser 32 bytes (64 hex chars), tiene %d bytes", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt cifra plaintext y devuelve [nonce || ciphertext || tag].
// Si plaintext está vacío, devuelve nil — no encriptar la cadena vacía.
func (c *Cipher) Encrypt(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return out, nil
}

// Decrypt descifra blob producido por Encrypt. Si blob es nil/vacío devuelve "".
func (c *Cipher) Decrypt(blob []byte) (string, error) {
	if len(blob) == 0 {
		return "", nil
	}
	ns := c.aead.NonceSize()
	if len(blob) < ns {
		return "", errors.New("crypto: blob demasiado corto")
	}
	nonce, ct := blob[:ns], blob[ns:]
	plain, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
