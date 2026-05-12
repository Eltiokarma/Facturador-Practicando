// Package users maneja persistencia y verificación de usuarios.
package users

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Rol string

const (
	RolDueno    Rol = "dueno"
	RolContador Rol = "contador"
	RolCajero   Rol = "cajero"
)

type User struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Email     string
	Nombre    string
	Rol       Rol
	Activo    bool
	CreatedAt time.Time
}

var ErrNotFound = errors.New("users: no encontrado")
var ErrInvalidCredentials = errors.New("users: credenciales inválidas")

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Create inserta un usuario nuevo y devuelve el ID.
func (s *Store) Create(ctx context.Context, tenantID uuid.UUID, email, nombre, password string, rol Rol) (uuid.UUID, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, err
	}
	id := uuid.New()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO users (id, tenant_id, email, password_hash, nombre, rol, activo)
		VALUES ($1,$2,$3,$4,$5,$6,TRUE)
	`, id, tenantID, strings.ToLower(strings.TrimSpace(email)), string(hash), nombre, string(rol))
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// Authenticate verifica email+password. Si OK devuelve el User.
func (s *Store) Authenticate(ctx context.Context, email, password string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var u User
	var hash string
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, nombre, rol, activo, created_at, password_hash
		FROM users
		WHERE lower(email) = $1
		LIMIT 1
	`, email).Scan(&u.ID, &u.TenantID, &u.Email, &u.Nombre, &u.Rol, &u.Activo, &u.CreatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !u.Activo {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return &u, nil
}

// ByID busca un usuario por ID (para resolver desde el JWT).
func (s *Store) ByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, nombre, rol, activo, created_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.TenantID, &u.Email, &u.Nombre, &u.Rol, &u.Activo, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
