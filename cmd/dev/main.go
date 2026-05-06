// Command dev runs CodeValdOrg locally against a local ArangoDB and without Cross.
// The Makefile's `make dev` target sources .env before exec so secrets stay
// out of the source tree.
package main

import (
	"log"
	"os"

	"github.com/aosanya/CodeValdOrg/internal/app"
	"github.com/aosanya/CodeValdOrg/internal/config"
)

func main() {
	setDefault("BIND_ADDR", ":9090")
	setDefault("ARANGO_ENDPOINTS", "http://localhost:8529")
	setDefault("ARANGO_USER", "root")
	setDefault("ARANGO_PASSWORD", "")
	setDefault("AGENCY_ID", "dev-agency")
	setDefault("ARANGO_DB_NAME", "agency-dev-agency")
	setDefault("CROSS_ENDPOINT", "")
	setDefault("ORG_ISSUER_URL", "http://localhost:9090")

	log.Println("codevaldorg[dev]: starting with local-dev defaults")
	if err := app.Run(config.Load()); err != nil {
		log.Fatalf("codevaldorg[dev]: %v", err)
	}
}

func setDefault(key, val string) {
	if _, ok := os.LookupEnv(key); !ok {
		os.Setenv(key, val)
	}
}
