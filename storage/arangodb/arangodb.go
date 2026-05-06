// Package arangodb implements the ArangoDB backend for CodeValdOrg.
// All implementation logic lives in SharedLib entitygraph/arangodb; this package
// is a thin service-scoped adapter that fixes the collection and graph names.
//
// Document collections:
//   - org_entities       — identity graph entities
//   - org_oauth_clients  — OAuth client registry
//   - org_oauth_artifacts — OAuth artifacts (TTL-indexed)
//   - org_audit_events   — append-only audit log
//
// Infrastructure collections:
//   - org_relationships      — ArangoDB EDGE collection
//   - org_schemas_draft      — mutable draft schema per agency
//   - org_schemas_published  — immutable published schema snapshots
//
// Named graph: org_graph
package arangodb

import (
	"fmt"

	driver "github.com/arangodb/go-driver"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
	sharedadb "github.com/aosanya/CodeValdSharedLib/entitygraph/arangodb"
	"github.com/aosanya/CodeValdSharedLib/types"
)

// Backend is a type alias for the shared ArangoDB Backend.
type Backend = sharedadb.Backend

// Config is the connection parameters for CodeValdOrg's ArangoDB backend.
type Config = sharedadb.ConnConfig

// toSharedConfig expands a CodeValdOrg Config into a full SharedLib Config,
// filling in the fixed org-specific collection and graph names.
func toSharedConfig(cfg Config) sharedadb.Config {
	return sharedadb.Config{
		Endpoint:            cfg.Endpoint,
		Username:            cfg.Username,
		Password:            cfg.Password,
		Database:            cfg.Database,
		Schema:              cfg.Schema,
		EntityCollection:    "org_entities",
		RelCollection:       "org_relationships",
		SchemasDraftCol:     "org_schemas_draft",
		SchemasPublishedCol: "org_schemas_published",
		GraphName:           "org_graph",
	}
}

// New constructs a Backend from an already-open driver.Database using the
// provided schema, ensures all collections and the named graph exist, and
// returns the Backend as both a DataManager and a SchemaManager.
func New(db driver.Database, schema types.Schema) (entitygraph.DataManager, entitygraph.SchemaManager, error) {
	if db == nil {
		return nil, nil, fmt.Errorf("arangodb: New: database must not be nil")
	}
	scfg := toSharedConfig(Config{Schema: schema})
	return sharedadb.New(db, scfg)
}

// NewBackend connects to ArangoDB using cfg, ensures all collections exist
// (including org_relationships as an edge collection), bootstraps the
// org_graph named graph, and returns a ready-to-use Backend.
func NewBackend(cfg Config) (*Backend, error) {
	if cfg.Database == "" {
		return nil, fmt.Errorf("arangodb: NewBackend: Database must be set")
	}
	scfg := toSharedConfig(cfg)
	return sharedadb.NewBackend(scfg)
}

// NewBackendFromDB constructs a Backend from an already-open driver.Database.
// Intended for tests that manage their own database lifecycle.
func NewBackendFromDB(db driver.Database, schema types.Schema) (*Backend, error) {
	if db == nil {
		return nil, fmt.Errorf("arangodb: NewBackendFromDB: database must not be nil")
	}
	scfg := toSharedConfig(Config{Schema: schema})
	return sharedadb.NewBackendFromDB(db, scfg)
}
