package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type ctxKey int

const ctxClaimsKey ctxKey = 1

// Middleware exige un access token válido en Authorization: Bearer.
// Si no, responde 401.
func Middleware(s *Signer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				writeUnauthorized(w, "falta header Authorization")
				return
			}
			tok := strings.TrimPrefix(h, "Bearer ")
			claims, err := s.Parse(tok, KindAccess)
			if err != nil {
				writeUnauthorized(w, "token inválido o expirado")
				return
			}
			ctx := context.WithValue(r.Context(), ctxClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromContext devuelve los claims del token autenticado.
func FromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(ctxClaimsKey).(*Claims)
	return c, ok
}

// UserIDFrom devuelve el UUID del usuario autenticado.
func UserIDFrom(ctx context.Context) (uuid.UUID, bool) {
	c, ok := FromContext(ctx)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(c.UserID)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// TenantIDFrom devuelve el UUID del tenant del usuario autenticado.
func TenantIDFrom(ctx context.Context) (uuid.UUID, bool) {
	c, ok := FromContext(ctx)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(c.TenantID)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized","detalle":"` + msg + `"}`))
}
