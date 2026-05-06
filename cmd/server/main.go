// Command server is the production CodeValdOrg gRPC microservice.
// Configuration is read strictly from environment variables — no .env is loaded.
package main

import (
	"log"

	"github.com/aosanya/CodeValdOrg/internal/app"
	"github.com/aosanya/CodeValdOrg/internal/config"
)

func main() {
	if err := app.Run(config.Load()); err != nil {
		log.Fatalf("codevaldorg: %v", err)
	}
}
