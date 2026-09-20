// Package config memuat seluruh konfigurasi dari environment.
//
// Semua nilai dibaca sekali saat start lalu diperlakukan read-only. Kalau ada
// yang wajib tapi kosong, aplikasi berhenti di sini, bukan meledak nanti saat
// request pertama masuk.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName string
	Env     string
	Port    string

	DB    DBConfig
	Redis RedisConfig
	JWT   JWTConfig

	Engine EngineConfig

	CORSAllowedOrigins []string
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN merangkai connection string MySQL. parseTime wajib true supaya kolom
// DATETIME masuk ke time.Time, bukan []byte.
func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_unicode_ci&multiStatements=true",
		d.User, d.Password, d.Host, d.Port, d.Name,
	)
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func (r RedisConfig) Addr() string {
	return net(r.Host, r.Port)
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
	// SessionTTL adalah umur sessionId di Redis. Dibuat terpisah dari
	// Expiration supaya sesi bisa lebih pendek dari umur JWT-nya.
	SessionTTL   time.Duration
	CookieName   string
	CookieSecure bool
}

type EngineConfig struct {
	// Dir adalah root folder engine, tempat main.py berada.
	Dir string
	// Python menunjuk ke interpreter venv runner engine.
	Python string
	// MaxConcurrent membatasi berapa run engine boleh jalan bersamaan.
	MaxConcurrent int
	Timeout       time.Duration
}

func net(host, port string) string { return host + ":" + port }

// Load membaca .env kalau ada, lalu environment. Environment selalu menang.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppName: env("APP_NAME", "geosquad-api"),
		Env:     env("APP_ENV", "development"),
		Port:    env("APP_PORT", "8084"),

		DB: DBConfig{
			Host:            env("DB_HOST", "localhost"),
			Port:            env("DB_PORT", "3306"),
			Name:            env("DB_NAME", "geosquad"),
			User:            env("DB_USER", "root"),
			Password:        os.Getenv("DB_PASSWORD"),
			MaxOpenConns:    envInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: envDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},

		Redis: RedisConfig{
			Host:     env("REDIS_HOST", "localhost"),
			Port:     env("REDIS_PORT", "6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       envInt("REDIS_DB", 0),
		},

		JWT: JWTConfig{
			Secret:       os.Getenv("JWT_SECRET"),
			Expiration:   envDuration("JWT_EXPIRATION", 24*time.Hour),
			SessionTTL:   envDuration("SESSION_TTL", 24*time.Hour),
			CookieName:   env("SESSION_COOKIE_NAME", "SID"),
			CookieSecure: envBool("SESSION_COOKIE_SECURE", false),
		},

		Engine: EngineConfig{
			Dir:           env("ENGINE_DIR", "../engine"),
			Python:        env("ENGINE_PYTHON", "../engine/.venv/bin/python"),
			MaxConcurrent: envInt("ENGINE_MAX_CONCURRENT", 2),
			Timeout:       envDuration("ENGINE_TIMEOUT", 60*time.Minute),
		},

		CORSAllowedOrigins: envList("CORS_ALLOWED_ORIGINS", []string{"http://localhost:4200"}),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	// Secret pendek bikin HMAC gampang di-brute force. 32 karakter itu lantai,
	// bukan target.
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET wajib diisi minimal 32 karakter (sekarang %d)", len(c.JWT.Secret))
	}
	if c.DB.Name == "" {
		return fmt.Errorf("DB_NAME wajib diisi")
	}
	if c.IsProduction() && !c.JWT.CookieSecure {
		return fmt.Errorf("SESSION_COOKIE_SECURE wajib true di production")
	}
	return nil
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Env, "production")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(key)); err == nil {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(key)); err == nil {
		return v
	}
	return fallback
}

func envList(key string, fallback []string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
