package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type SunatMode string

const (
	SunatModeBeta SunatMode = "beta"
	SunatModeProd SunatMode = "prod"
)

type Config struct {
	Port            string
	SunatMode       SunatMode
	JWTSecret       string
	JWTAccessTTL    time.Duration
	JWTRefreshTTL   time.Duration
	CORSOrigins     []string
	PostgresDSN     string
	RedisAddr       string
	MotorURL        string
	DataDir         string
}

func Load() (*Config, error) {
	c := &Config{
		Port:        getEnv("API_PORT", "8080"),
		SunatMode:   SunatMode(strings.ToLower(getEnv("SUNAT_MODE", "beta"))),
		JWTSecret:   os.Getenv("API_JWT_SECRET"),
		CORSOrigins: splitCSV(getEnv("API_CORS_ORIGINS", "http://localhost:5173")),
		MotorURL:    getEnv("MOTOR_URL", "http://motor:8000"),
		DataDir:     getEnv("DATA_DIR", "/app/data"),
	}

	if c.SunatMode != SunatModeBeta && c.SunatMode != SunatModeProd {
		return nil, fmt.Errorf("SUNAT_MODE inválido: %q (esperaba 'beta' o 'prod')", c.SunatMode)
	}
	if len(c.JWTSecret) < 32 {
		return nil, errors.New("API_JWT_SECRET debe tener al menos 32 caracteres")
	}

	var err error
	c.JWTAccessTTL, err = time.ParseDuration(getEnv("API_JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("API_JWT_ACCESS_TTL inválido: %w", err)
	}
	c.JWTRefreshTTL, err = time.ParseDuration(getEnv("API_JWT_REFRESH_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("API_JWT_REFRESH_TTL inválido: %w", err)
	}

	c.PostgresDSN = fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getEnv("POSTGRES_USER", "facturador"),
		getEnv("POSTGRES_PASSWORD", ""),
		getEnv("POSTGRES_HOST", "postgres"),
		getEnv("POSTGRES_PORT", "5432"),
		getEnv("POSTGRES_DB", "facturador"),
	)
	c.RedisAddr = fmt.Sprintf("%s:%s", getEnv("REDIS_HOST", "redis"), getEnv("REDIS_PORT", "6379"))

	return c, nil
}

func getEnv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
