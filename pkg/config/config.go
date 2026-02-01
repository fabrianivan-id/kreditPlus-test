package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppAddr          string
	DBDSN            string
	APIKey           string
	RateLimitPerMin  int
	BodyLimitBytes   int64
	AllowedOrigins   []string
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
	DBMaxOpenConns   int
	DBMaxIdleConns   int
	DBConnMaxLife    time.Duration
}

func Load() Config {
	return Config{
		AppAddr:         getEnv("APP_ADDR", ":8080"),
		DBDSN:           getEnv("DB_DSN", "root:password@tcp(localhost:3306)/kreditplus?parseTime=true"),
		APIKey:          os.Getenv("API_KEY"),
		RateLimitPerMin: getEnvInt("RATE_LIMIT_PER_MIN", 60),
		BodyLimitBytes:  getEnvInt64("BODY_LIMIT_BYTES", 1<<20),
		AllowedOrigins:  splitEnv("ALLOWED_ORIGINS"),
		ReadTimeout:     getEnvDuration("READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		DBMaxOpenConns:  getEnvInt("DB_MAX_OPEN_CONNS", 10),
		DBMaxIdleConns:  getEnvInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLife:   getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
	}
}

func getEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return def
}

func getEnvInt64(key string, def int64) int64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
	}
	return def
}

func splitEnv(key string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
