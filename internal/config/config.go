// Package config loads CodeValdOrg runtime configuration from environment variables.
package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aosanya/CodeValdSharedLib/serverutil"
)

// Config holds all runtime configuration for the CodeValdOrg service.
type Config struct {
	// Required
	AgencyID        string
	ArangoEndpoints []string
	ArangoUser      string
	ArangoPassword  string
	CrossEndpoint   string
	IssuerURL       string

	// Optional with defaults
	ArangoDBName      string
	BindAddr          string
	MetricsAddr       string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	AuthCodeTTL       time.Duration
	ClientSecretGrace time.Duration
	Argon2Time        uint32
	Argon2MemoryKiB   uint32
	Argon2Threads     uint8
	RegistrarInterval time.Duration
	PingTimeout       time.Duration
	LogLevel          string
}

// Load reads environment variables, performs two-pass validation, and returns
// a Config. Exits with code 2 if any required variable is missing.
func Load() Config {
	// Pass 1: required variables.
	required := map[string]string{
		"AGENCY_ID":        os.Getenv("AGENCY_ID"),
		"ARANGO_ENDPOINTS": os.Getenv("ARANGO_ENDPOINTS"),
		"ARANGO_USER":      os.Getenv("ARANGO_USER"),
		"ARANGO_PASSWORD":  os.Getenv("ARANGO_PASSWORD"),
		"CROSS_ENDPOINT":   os.Getenv("CROSS_ENDPOINT"),
		"ORG_ISSUER_URL":   os.Getenv("ORG_ISSUER_URL"),
	}
	var missing []string
	for k, v := range required {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "codevaldorg: missing required env vars: %s\n", strings.Join(missing, ", "))
		os.Exit(2)
	}

	agencyID := required["AGENCY_ID"]
	arangoDBName := serverutil.EnvOrDefault("ARANGO_DB_NAME", "agency-"+agencyID)

	endpoints := strings.Split(required["ARANGO_ENDPOINTS"], ",")
	for i, ep := range endpoints {
		endpoints[i] = strings.TrimSpace(ep)
	}

	cfg := Config{
		AgencyID:          agencyID,
		ArangoEndpoints:   endpoints,
		ArangoUser:        required["ARANGO_USER"],
		ArangoPassword:    required["ARANGO_PASSWORD"],
		CrossEndpoint:     required["CROSS_ENDPOINT"],
		IssuerURL:         required["ORG_ISSUER_URL"],
		ArangoDBName:      arangoDBName,
		BindAddr:          serverutil.EnvOrDefault("BIND_ADDR", ":50058"),
		MetricsAddr:       serverutil.EnvOrDefault("METRICS_ADDR", ":9091"),
		AccessTokenTTL:    serverutil.ParseDurationString("ORG_ACCESS_TOKEN_TTL", time.Hour),
		RefreshTokenTTL:   serverutil.ParseDurationString("ORG_REFRESH_TOKEN_TTL", 720*time.Hour),
		AuthCodeTTL:       serverutil.ParseDurationString("ORG_AUTH_CODE_TTL", 60*time.Second),
		ClientSecretGrace: serverutil.ParseDurationString("ORG_CLIENT_SECRET_GRACE", 5*time.Minute),
		Argon2Time:        parseUint32("ORG_ARGON2_TIME", 3),
		Argon2MemoryKiB:   parseUint32("ORG_ARGON2_MEMORY_KIB", 65536),
		Argon2Threads:     parseUint8("ORG_ARGON2_THREADS", 4),
		RegistrarInterval: serverutil.ParseDurationString("ORG_REGISTRAR_INTERVAL", 20*time.Second),
		PingTimeout:       serverutil.ParseDurationString("CROSS_PING_TIMEOUT", 5*time.Second),
		LogLevel:          serverutil.EnvOrDefault("LOG_LEVEL", "info"),
	}

	// Pass 2: range/format checks.
	if cfg.Argon2Time < 1 {
		log.Fatal("codevaldorg: ORG_ARGON2_TIME must be >= 1")
	}
	if cfg.Argon2MemoryKiB < 8192 {
		log.Fatal("codevaldorg: ORG_ARGON2_MEMORY_KIB must be >= 8192")
	}
	if cfg.Argon2Threads < 1 {
		log.Fatal("codevaldorg: ORG_ARGON2_THREADS must be >= 1")
	}
	if cfg.AccessTokenTTL <= 0 {
		log.Fatal("codevaldorg: ORG_ACCESS_TOKEN_TTL must be positive")
	}
	if cfg.RefreshTokenTTL <= 0 {
		log.Fatal("codevaldorg: ORG_REFRESH_TOKEN_TTL must be positive")
	}
	if cfg.AuthCodeTTL <= 0 {
		log.Fatal("codevaldorg: ORG_AUTH_CODE_TTL must be positive")
	}

	return cfg
}

func parseUint32(key string, def uint32) uint32 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	var n uint32
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		log.Printf("codevaldorg: %s=%q invalid — using default %d", key, v, def)
		return def
	}
	return n
}

func parseUint8(key string, def uint8) uint8 {
	v := parseUint32(key, uint32(def))
	return uint8(v)
}
