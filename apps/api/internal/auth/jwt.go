// Package auth implementa emisión y validación de tokens JWT.
//
// Dos tipos:
//   - access:  vida corta (~15m), va en Authorization: Bearer <token>.
//   - refresh: vida larga (~30d), se usa solo contra /auth/refresh.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenKind string

const (
	KindAccess  TokenKind = "access"
	KindRefresh TokenKind = "refresh"
)

type Claims struct {
	Kind     TokenKind `json:"knd"`
	UserID   string    `json:"uid"`
	TenantID string    `json:"tid"`
	Rol      string    `json:"rol,omitempty"`
	jwt.RegisteredClaims
}

type Signer struct {
	Secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func NewSigner(secret string, accessTTL, refreshTTL time.Duration) *Signer {
	return &Signer{Secret: []byte(secret), AccessTTL: accessTTL, RefreshTTL: refreshTTL}
}

func (s *Signer) issue(kind TokenKind, userID, tenantID uuid.UUID, rol string, ttl time.Duration) (string, error) {
	now := time.Now()
	c := Claims{
		Kind:     kind,
		UserID:   userID.String(),
		TenantID: tenantID.String(),
		Rol:      rol,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString(s.Secret)
}

func (s *Signer) Access(userID, tenantID uuid.UUID, rol string) (string, error) {
	return s.issue(KindAccess, userID, tenantID, rol, s.AccessTTL)
}

func (s *Signer) Refresh(userID, tenantID uuid.UUID) (string, error) {
	return s.issue(KindRefresh, userID, tenantID, "", s.RefreshTTL)
}

var ErrInvalidToken = errors.New("auth: token inválido")

func (s *Signer) Parse(token string, want TokenKind) (*Claims, error) {
	var c Claims
	parsed, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.Secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if c.Kind != want {
		return nil, ErrInvalidToken
	}
	return &c, nil
}
