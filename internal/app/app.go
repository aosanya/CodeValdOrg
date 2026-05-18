// Package app holds the shared runtime wiring for CodeValdOrg. Both the
// production binary (cmd/server) and the local dev binary (cmd/dev) call
// Run; they differ only in which environment variables they set before
// loading config.
package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"

	codevaldorg "github.com/aosanya/CodeValdOrg"
	pb "github.com/aosanya/CodeValdOrg/gen/go/codevaldorg/v1"
	"github.com/aosanya/CodeValdOrg/internal/config"
	"github.com/aosanya/CodeValdOrg/internal/httphandler"
	"github.com/aosanya/CodeValdOrg/internal/registrar"
	"github.com/aosanya/CodeValdOrg/internal/server"
	orgdb "github.com/aosanya/CodeValdOrg/storage/arangodb"
	"github.com/aosanya/CodeValdSharedLib/entitygraph"
	healthpb "github.com/aosanya/CodeValdSharedLib/gen/go/codevaldhealth/v1"
	"github.com/aosanya/CodeValdSharedLib/health"
	"github.com/aosanya/CodeValdSharedLib/serverutil"
)

// Run starts all CodeValdOrg subsystems and blocks until SIGINT/SIGTERM.
func Run(cfg config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Signal handling ───────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-quit
		log.Println("codevaldorg: shutdown signal received")
		cancel()
	}()

	// Use the first ArangoDB endpoint.
	endpoint := cfg.ArangoEndpoints[0]

	// ── ArangoDB backend ──────────────────────────────────────────────────────
	backend, err := orgdb.NewBackend(orgdb.Config{
		Endpoint: endpoint,
		Username: cfg.ArangoUser,
		Password: cfg.ArangoPassword,
		Database: cfg.ArangoDBName,
		Schema:   codevaldorg.DefaultOrgSchema(),
	})
	if err != nil {
		return fmt.Errorf("ArangoDB backend: %w", err)
	}

	// ── Schema seed (idempotent on startup) ───────────────────────────────────
	seedCtx, seedCancel := context.WithTimeout(ctx, 30*time.Second)
	if err := entitygraph.SeedSchema(seedCtx, backend, cfg.AgencyID, codevaldorg.DefaultOrgSchema()); err != nil {
		log.Printf("codevaldorg: schema seed: %v", err)
	}
	seedCancel()

	// ── Cross registrar (optional) ────────────────────────────────────────────
	var pub codevaldorg.CrossPublisher
	if cfg.CrossEndpoint != "" {
		reg, regErr := registrar.New(
			cfg.CrossEndpoint,
			cfg.AdvertiseAddr,
			cfg.AgencyID,
			cfg.RegistrarInterval,
			cfg.PingTimeout,
		)
		if regErr != nil {
			log.Printf("codevaldorg: registrar: %v — continuing without registration", regErr)
		} else {
			pub = reg
			defer reg.Close()
			go reg.Run(ctx)
		}
	} else {
		log.Println("codevaldorg: CROSS_ENDPOINT not set — skipping Cross registration")
	}

	// ── OrgManager (constructed after registrar so publisher is available) ──────
	mgr := codevaldorg.NewOrgManager(backend, backend, pub, codevaldorg.NewClock(), codevaldorg.ManagerConfig{
		AgencyID:          cfg.AgencyID,
		IssuerURL:         cfg.IssuerURL,
		AccessTokenTTL:    cfg.AccessTokenTTL,
		RefreshTokenTTL:   cfg.RefreshTokenTTL,
		AuthCodeTTL:       cfg.AuthCodeTTL,
		ClientSecretGrace: cfg.ClientSecretGrace,
		Argon2Time:        cfg.Argon2Time,
		Argon2MemoryKiB:   cfg.Argon2MemoryKiB,
		Argon2Threads:     cfg.Argon2Threads,
	})

	// ── gRPC server ───────────────────────────────────────────────────────────
	grpcServer, _ := serverutil.NewGRPCServer()
	pb.RegisterOrgServiceServer(grpcServer, server.New(mgr, cfg.AgencyID))
	healthpb.RegisterHealthServiceServer(grpcServer, health.New("codevaldorg"))

	// ── HTTP handler (OAuth endpoints + healthz) ──────────────────────────────
	httpHandler := httphandler.New(cfg.AgencyID, cfg.IssuerURL, mgr)
	httpServer := &http.Server{
		Handler:      httpHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// ── TCP listener + cmux ───────────────────────────────────────────────────
	lis, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.BindAddr, err)
	}

	mux := cmux.New(lis)
	grpcLis := mux.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"),
	)
	httpLis := mux.Match(cmux.Any())

	go func() {
		if err := grpcServer.Serve(grpcLis); err != nil && err != grpc.ErrServerStopped {
			log.Printf("codevaldorg: gRPC server error: %v", err)
		}
	}()
	go func() {
		if err := httpServer.Serve(httpLis); err != nil && err != http.ErrServerClosed {
			log.Printf("codevaldorg: HTTP server error: %v", err)
		}
	}()
	go func() {
		if err := mux.Serve(); err != nil {
			log.Printf("codevaldorg: cmux serve error: %v", err)
		}
	}()

	log.Printf("codevaldorg: listening on %s (agency=%s, gRPC + HTTP via cmux)", cfg.BindAddr, cfg.AgencyID)

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	<-ctx.Done()
	log.Println("codevaldorg: shutting down")

	grpcServer.GracefulStop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("codevaldorg: HTTP shutdown error: %v", err)
	}

	log.Println("codevaldorg: stopped")
	return nil
}
