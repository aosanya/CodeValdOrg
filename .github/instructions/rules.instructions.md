````instructions
---
applyTo: '**'
---

# CodeValdOrg — Code Structure Rules

## Service Design Principles

CodeValdOrg is a **Go gRPC microservice** that owns identity, authentication,
and authorization for the CodeVald platform — not a library and not a monolith.
These rules reflect that:

- **Has a `cmd/main.go` binary entry point** — wires all dependencies and starts the server
- **No business logic in `cmd/`** — `main.go` only constructs dependencies and calls `server.Run`
- **Callers inject dependencies** — `Backend`, `KeyHasher`, and any cross-service
  clients are never hardcoded
- **Exported API surface is minimal** — expose only what other packages within this module need
- **No agency / task / git / AI / comms domain logic** — this service owns
  principals, keys, and scopes only

---

## Interface-First Design

**Always define interfaces before concrete types.**

```go
// ✅ CORRECT — interface at root package level; concrete impl is unexported in internal/manager/
type OrgManager interface {
    IssueKey(ctx context.Context, req IssueKeyRequest) (Key, string, error) // returns plaintext ONCE
    RevokeKey(ctx context.Context, keyID string) error
    RotateKey(ctx context.Context, keyID string) (Key, string, error)
    GetKey(ctx context.Context, keyID string) (Key, error) // metadata only, no plaintext
    ListKeys(ctx context.Context, filter KeyFilter) ([]Key, error)

    VerifyToken(ctx context.Context, token string) (Principal, error)
    Authorize(ctx context.Context, req AuthorizeRequest) (Decision, error)
}

// ❌ WRONG — leaking a concrete storage struct to callers
type ArangoOrgManager struct {
    db driver.Database
}
```

**File layout — one primary concern per file:**

```
errors.go                            → ErrKeyNotFound, ErrKeyRevoked, ErrInvalidScope, ErrUnauthorized
models.go                            → Key, IssueKeyRequest, Principal, Scope, AuthorizeRequest, Decision
internal/manager/manager.go          → Concrete OrgManager implementation
internal/server/server.go            → Inbound gRPC server (OrgService handlers)
internal/config/config.go            → Configuration struct + loader
internal/crypto/hasher.go            → KeyHasher implementation (bcrypt / argon2)
storage/arangodb/storage.go          → Config, Backend struct, constructors, collection setup
storage/arangodb/docs.go             → ArangoDB document types and domain↔document conversions
storage/arangodb/ops.go              → Backend interface method implementations
cmd/main.go                          → Dependency wiring only
```

---

## Key-Issuance Rules

**Key issuance is the most security-sensitive operation in this service.**

```go
// ✅ CORRECT — hash before persistence, return plaintext ONCE, publish event
func (m *manager) IssueKey(ctx context.Context, req IssueKeyRequest) (Key, string, error) {
    plaintext, err := m.rng.NewToken()
    if err != nil {
        return Key{}, "", err
    }
    hash, err := m.hasher.Hash(plaintext)
    if err != nil {
        return Key{}, "", err
    }
    key, err := m.backend.Insert(ctx, req, hash) // only the hash is persisted
    if err != nil {
        return Key{}, "", err
    }
    // MANDATORY: publish so caches in other services invalidate
    m.crossClient.Publish(ctx, "org.token.issued", key.ID)
    return key, plaintext, nil // plaintext returned ONCE to caller, never again
}

// ❌ WRONG — storing plaintext, leaking it on read, skipping event publish
func (m *manager) IssueKey(ctx context.Context, req IssueKeyRequest) (Key, string, error) {
    plaintext, _ := m.rng.NewToken()
    key, _ := m.backend.Insert(ctx, req, plaintext) // plaintext hits storage
    return key, plaintext, nil                      // no event published
}
```

**Rules for any code path that touches a plaintext key or token:**

- Plaintext MUST NOT be logged — no `log.Printf("token=%s", t)`, no debug print
- Plaintext MUST NOT be persisted — only the hash
- Plaintext MUST NOT be returned by `GetKey` / `ListKeys` — metadata only
- Plaintext MUST be returned from `IssueKey` / `RotateKey` exactly once

---

## gRPC Handler Rules

**Handlers are thin — delegate immediately to `OrgManager`.**

```go
// ✅ CORRECT — handler delegates to interface
func (s *server) IssueKey(ctx context.Context, req *pb.IssueKeyRequest) (*pb.IssueKeyResponse, error) {
    key, plaintext, err := s.manager.IssueKey(ctx, toModel(req))
    if err != nil {
        return nil, toGRPCError(err)
    }
    return toIssueResponse(key, plaintext), nil
}

// ❌ WRONG — business logic inside handler
func (s *server) IssueKey(ctx context.Context, req *pb.IssueKeyRequest) (*pb.IssueKeyResponse, error) {
    // don't put hashing, storage, or pub/sub here
    h, _ := bcrypt.GenerateFromPassword(...)
    doc, _ := s.db.Collection("keys").CreateDocument(ctx, ...)
    ...
}
```

---

## Storage Backend Rules

The `Backend` interface is the injection point. The caller (`cmd/main.go`)
constructs the desired `Backend` and passes it to `NewOrgManager`. The root
package and `internal/manager/` never import ArangoDB drivers directly.

```go
// Backend interface — defined in root package or internal/manager/
type Backend interface {
    Insert(ctx context.Context, req IssueKeyRequest, hash []byte) (Key, error)
    Get(ctx context.Context, keyID string) (Key, error)
    FindByHash(ctx context.Context, hash []byte) (Key, error) // used by VerifyToken
    Revoke(ctx context.Context, keyID string) error
    List(ctx context.Context, filter KeyFilter) ([]Key, error)
}

// ✅ CORRECT — Backend injected by cmd/main.go
b, _ := arangodb.NewBackend(cfg.ArangoDB)
mgr := manager.NewOrgManager(b, hasher, rng, crossClient)

// ❌ WRONG — hardcoded driver inside manager
func NewOrgManager() OrgManager {
    db, _ := arangodb.NewDatabase(...)
    return &orgManager{db: db}
}
```

---

## Authorization Rules

**`Authorize` is called by every other CodeVald service on every authenticated
request. It must be deterministic, side-effect-free, and fast.**

```go
// ✅ CORRECT — pure decision, no writes, explicit inputs
func (m *manager) Authorize(ctx context.Context, req AuthorizeRequest) (Decision, error) {
    p, err := m.backend.GetPrincipal(ctx, req.PrincipalID)
    if err != nil {
        return Decision{}, err
    }
    if !p.Scopes.Allows(req.Scope, req.Resource) {
        return Decision{Allowed: false, Reason: "scope not granted"}, nil
    }
    return Decision{Allowed: true}, nil
}

// ❌ WRONG — mutating state inside Authorize, or performing cross-service calls
func (m *manager) Authorize(ctx context.Context, req AuthorizeRequest) (Decision, error) {
    m.backend.RecordAuditLog(ctx, req)         // writes in a read-path RPC
    m.crossClient.CheckSomething(ctx, req)     // dependency on another service
    ...
}
```

Audit logging is allowed but MUST happen asynchronously (enqueued on a channel)
so `Authorize` latency is not gated on disk writes.

---

## CodeValdCross Registration Rules

**Registration must happen on startup and repeat as a liveness heartbeat.**

```go
// ✅ CORRECT — register on startup with heartbeat loop
func register(ctx context.Context, crossAddr string) {
    req := &pb.RegisterRequest{
        ServiceName: "codevaldorg",
        Addr:        ":50051",
        Produces:    []string{"org.token.issued", "org.token.revoked"},
        Consumes:    []string{},
        Routes:      orgRoutes(),
    }
    for {
        if err := crossClient.Register(ctx, req); err != nil {
            log.Printf("codevaldorg: register error: %v", err)
        }
        select {
        case <-ctx.Done():
            return
        case <-time.After(20 * time.Second):
        }
    }
}

// ❌ WRONG — register once and forget (Cross will drop the service after timeout)
func main() {
    crossClient.Register(ctx, req)
    server.Run(ctx)
}
```

---

## Error Types

All exported errors live in `errors.go`. Never scatter sentinel errors across files.

```go
// errors.go — all exported error types
var (
    ErrKeyNotFound     = errors.New("key not found")
    ErrKeyRevoked      = errors.New("key is revoked")
    ErrInvalidScope    = errors.New("invalid or unknown scope")
    ErrUnauthorized    = errors.New("caller is not authorized for the requested scope")
    ErrTokenExpired    = errors.New("token has expired")
)
```

Map errors to gRPC status codes in the server layer, not in the manager:

```go
// internal/server/server.go
func toGRPCError(err error) error {
    switch {
    case errors.Is(err, ErrKeyNotFound):
        return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, ErrKeyRevoked), errors.Is(err, ErrTokenExpired):
        return status.Error(codes.Unauthenticated, err.Error())
    case errors.Is(err, ErrUnauthorized):
        return status.Error(codes.PermissionDenied, err.Error())
    case errors.Is(err, ErrInvalidScope):
        return status.Error(codes.InvalidArgument, err.Error())
    default:
        return status.Error(codes.Internal, err.Error())
    }
}
```

---

## Context & Cancellation Rules

- Every public method takes `context.Context` as the first argument
- Check `ctx.Err()` in loops (heartbeat, retry loops, audit flushers)
- Pass `ctx` to all storage calls and cross-service calls
- Never use `context.Background()` inside library code — accept context from caller

---

## Secret-Handling Rules

- **Never log plaintext keys, tokens, or passwords** — redact before logging
- **Never include plaintext secrets in error messages** — return `ErrUnauthorized`
  without echoing the bad input
- **Compare hashes in constant time** — use `subtle.ConstantTimeCompare` or the
  hasher's built-in `Compare`
- **Zero sensitive byte slices after use** where practical
- **Configuration secrets** (signing keys, DB passwords) come from env vars or
  secret managers, never from checked-in YAML defaults

---

## Naming Conventions

| Category | Convention | Example |
|---|---|---|
| Branch | `feature/ORG-XXX_description` | `feature/ORG-001_issue-key` |
| Commit | `ORG-XXX: message` | `ORG-001: Add IssueKey gRPC handler` |
| Package | lowercase, no abbreviations | `org`, `manager`, `server`, `crypto` |
| Interfaces | noun-only | `OrgManager`, `Backend`, `KeyHasher` |
| Exported types | PascalCase | `Key`, `IssueKeyRequest`, `Principal`, `Scope` |
| gRPC service | `OrgService` | in `proto/codevaldorg/org.proto` |

---

## Anti-Patterns

- ❌ **Agency / task / git / AI / comms domain logic** — not in this service
- ❌ **Frontend routes or HTML templates** — belongs in the frontend services
- ❌ **Storing plaintext keys or tokens** — hash before persistence, always
- ❌ **Logging plaintext secrets** — even in debug builds
- ❌ **Returning plaintext from read endpoints** — only `Issue` / `Rotate` return plaintext, once
- ❌ **Non-constant-time secret comparison** — use `subtle.ConstantTimeCompare`
- ❌ **Pub/sub topic strings as raw literals** — define as constants
- ❌ **Business logic in gRPC handlers** — delegate to `OrgManager`
- ❌ **Hardcoded ArangoDB connection in manager** — inject `Backend`
- ❌ **Skipping `org.token.issued` / `org.token.revoked`** — always publish on issue / revoke
````
