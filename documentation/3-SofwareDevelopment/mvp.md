# MVP — Active Task Backlog

## Overview
- **Objective**: Deliver CodeValdOrg as a production-ready standalone gRPC microservice providing OAuth 2.0 identity, token issuance, introspection, revocation, scope management, role/membership management, and OAuth client management.
- **Research baseline**: All 13 research areas closed (2026-04-27) — see [`research-gaps.md`](research-gaps.md)
- **Completed tasks**: see [`mvp_done.md`](mvp_done.md)
- **Detailed specs**: see [`mvp-details/`](mvp-details/)

## Workflow

### Completion Process (MANDATORY)
1. Implement and validate (`go build ./...`, `go vet ./...`, `go test -race ./...`)
2. Add row to `mvp_done.md`
3. Remove task from this file
4. Mark dependency references as `~~ORG-XXX~~ ✅`
5. Merge feature branch to main and delete it

### Branch Management
```bash
git checkout -b feature/ORG-XXX_description
# implement + validate
git checkout main
git merge feature/ORG-XXX_description --no-ff
git branch -d feature/ORG-XXX_description
```

### Status Legend
- 📋 **Not Started** — ready to begin (dependencies met)
- 🚀 **In Progress** — currently being worked on
- ⏸️ **Blocked** — waiting on dependencies

---

## P0: Foundation (unblocked — start here)

### ORG-001 — Module Scaffolding

| Task | Status | Depends On |
|------|--------|------------|
| ORG-001: `go.mod` + directory skeleton + stub `cmd/server/main.go` + `Makefile` + `Dockerfile.server` | 📋 Not Started | — |

**Scope**: Establish the Go module (`github.com/aosanya/CodeValdOrg`), directory tree
(`cmd/server/`, `internal/`, `proto/codevaldorg/v1/`, `gen/go/codevaldorg/v1/`, `storage/arangodb/`),
a stub `main.go` that compiles but does nothing, a `buf.yaml` + `buf.gen.yaml` for proto codegen,
and a `Makefile` with `build`, `test`, `lint`, `proto` targets matching CodeValdGit's conventions.

---

### ORG-002 — `schema.go` — `DefaultOrgSchema()`

| Task | Status | Depends On |
|------|--------|------------|
| ORG-002: Translate schema-reference.md into `DefaultOrgSchema()` at repo root | 📋 Not Started | ORG-001 |

**Scope**: All 15 TypeDefinitions across three storage collections:
- `org_principals` — `Organization`, `User`, `PasswordCredential`, `Role`, `Scope`, `Membership`, `Invitation`
- `org_oauth_artifacts` — `OAuthClient`, `ClientSecret`, `RedirectURI`, `AuthorizationCode`, `AccessToken`, `RefreshToken`, `TokenRevocation`
- `org_admin` — `AuditEvent`

All indexes (TTL on `org_oauth_artifacts.expires_at`, unique compounds on `(agency_id, email)`,
`(agency_id, name)`, `(agency_id, client_id)`, `(agency_id, token_hash)`), auto-inverse edge
names, immutability flags, and the built-in-role seeding flow.

See: [mvp-details/schema-reference.md](mvp-details/schema-reference.md)

---

### ORG-003 — `models.go` — Go value types

| Task | Status | Depends On |
|------|--------|------------|
| ORG-003: Go structs + enums mirroring all 15 TypeDefinitions | 📋 Not Started | ORG-001 |

**Scope**: One Go struct per TypeDefinition. All enum constants (`GrantType`, `TokenStatus`,
`MembershipRole`, `InvitationStatus`, `ClientType`). Request/response types for `OrgManager`
interface methods. `Clock` interface for testability (per testing-strategy.md).

See: [mvp-details/data-model/](mvp-details/data-model/)

---

### ORG-004 — `errors.go` — sentinel errors

| Task | Status | Depends On |
|------|--------|------------|
| ORG-004: All sentinel errors from error-catalog.md | 📋 Not Started | ORG-001 |

**Scope**: Every sentinel in the error catalog (`ErrOrganizationNotFound`, `ErrUserNotFound`,
`ErrInvalidCredentials`, `ErrTokenExpired`, `ErrTokenRevoked`, `ErrScopeNameCollision`,
`ErrInvalidScope`, `ErrClientNotFound`, `ErrInvalidClientSecret`, `ErrCodeVerifierMismatch`,
`ErrRefreshTokenReuse`, `ErrTemporarilyUnavailable`, and all others).
`internal/server/mappers.go` is the only place sentinels are translated to gRPC codes / HTTP
status / OAuth error bodies.

See: [mvp-details/error-catalog.md](mvp-details/error-catalog.md)

---

### ORG-005 — Proto + codegen

| Task | Status | Depends On |
|------|--------|------------|
| ORG-005: `proto/codevaldorg/v1/org.proto` — full RPC definition + `buf generate` | 📋 Not Started | ORG-001 |

**Scope**: All RPCs per grpc-api.md: organization lifecycle, user management, membership,
OAuth client, scope, auth-code grant, client-credentials grant, introspect, revoke,
list audit events. Message conventions: `agency_id` always field 1, cursor pagination
(`page_size` + `page_token`), `google.protobuf.Timestamp` on the wire. `buf generate`
produces Go stubs into `gen/go/codevaldorg/v1/`.

See: [mvp-details/grpc-api.md](mvp-details/grpc-api.md)

---

## P0: Core Interface + Config (depends on ORG-001)

### ORG-006 — `OrgManager` interface

| Task | Status | Depends On |
|------|--------|------------|
| ORG-006: Flat `OrgManager` interface + `orgManager` struct + stub implementations | 📋 Not Started | ORG-003, ORG-004 |

**Scope**: Single flat interface covering all domain operations — organization lifecycle,
user management, membership, invitation, OAuth client management, scope registration,
token issuance (auth code + PKCE + client credentials), introspection, revocation.
`orgManager` struct holds `dm entitygraph.DataManager`, `pub CrossPublisher`, `clock Clock`.
Stub implementations that return `ErrTemporarilyUnavailable` for all methods.

---

### ORG-007 — `internal/config/config.go`

| Task | Status | Depends On |
|------|--------|------------|
| ORG-007: `Config` struct + `Load()` — 6 required + 12 optional env vars | 📋 Not Started | ORG-001 |

**Scope**: Two-pass startup validation: required-set check then range/format check;
misconfigured process never reaches a half-started state. `AGENCY_ID` baked in at startup;
rejects RPCs whose `agencyId` doesn't match. Full env var table in configuration.md.

See: [mvp-details/configuration.md](mvp-details/configuration.md)

---

### ORG-008 — ArangoDB entitygraph backend

| Task | Status | Depends On |
|------|--------|------------|
| ORG-008: Thin adapter over `SharedLib` `entitygraph/arangodb`; fixed collection names | 📋 Not Started | ORG-002 |

**Scope**: `Backend = sharedadb.Backend`, `Config = sharedadb.ConnConfig`, `toSharedConfig()`
fills fixed collection/graph names (`org_principals`, `org_oauth_artifacts`, `org_admin`,
`org_relationships`, `org_schemas_draft`, `org_schemas_published`). Matches the CodeValdGit
adapter pattern. Integration tests skip without `ORG_ARANGO_ENDPOINT`.

---

## P1: Domain Logic (depends on ORG-006 + ORG-008)

### ORG-009 — Organization + User lifecycle

| Task | Status | Depends On |
|------|--------|------------|
| ORG-009: `CreateOrganization`, `GetOrganization`, `DisableOrganization`, `CreateUser`, `GetUser`, `UpdateUser`, `SuspendUser`, `DeleteUser` | 📋 Not Started | ORG-006, ORG-008 |

**Scope**: `SuspendUser` and `DisableOrganization` are bulk-revocation triggers — synchronous
chunked revoke before state change. `DeleteUser` soft-deletes; references preserved for audit.

---

### ORG-010 — Token issuance

| Task | Status | Depends On |
|------|--------|------------|
| ORG-010: Authorization code grant + PKCE (RFC 7636), client credentials grant, access token mint, refresh token rotation | 📋 Not Started | ORG-009, ORG-012, ORG-013 |

**Scope**: Prefixed tokens (`cv_ac_…` / `cv_at_…` / `cv_rt_…`), 256-bit `crypto/rand` tail,
SHA-256 hash at rest of full prefixed plaintext. Strong-CP publish-or-roll-back contract:
DB write + Cross publish must both succeed; any failure → 503 `temporarily_unavailable`.
Effective scope = strict intersection of `requested ∩ user grants ∩ client.allowedScopes ∩
{not deprecated}` bound to token via `has_scope` edges and frozen at mint time.
Refresh-rotation reuse detection walks the `parent` chain and revokes all ancestors +
descendants on detected reuse.

See: [mvp-details/token-issuance.md](mvp-details/token-issuance.md)

---

### ORG-011 — Token introspection

| Task | Status | Depends On |
|------|--------|------------|
| ORG-011: Hash lookup, `{active: false}` parity, caller auth, opt-in caching | 📋 Not Started | ORG-010 |

**Scope**: Lookup by `(agency_id, token_hash)` unique index — no `crypto/subtle` compare needed.
`{active: false}` returned for unknown / expired / revoked / malformed tokens (RFC 7662 parity).
Caller authentication required; anonymous probing rejected. Opt-in caching for v1 with mandatory
invalidation on `cross.org.{agencyID}.token.revoked`.

See: [mvp-details/introspection.md](mvp-details/introspection.md)

---

### ORG-012 — Token revocation + pub/sub

| Task | Status | Depends On |
|------|--------|------------|
| ORG-012: Revoke by hash, publish `cross.org.{agencyID}.token.revoked`, bulk revocation | 📋 Not Started | ORG-008 |

**Scope**: At-least-once revocation via Cross event bus. `TokenRevocation.expires_at` matches
underlying token's natural expiry so the TTL purges revocation records exactly when they stop
being load-bearing. Race on refresh-rotation reuse resolved by unique index on
`(agency_id, token_hash)` — no locking needed.

See: [mvp-details/revocation-and-cache.md](mvp-details/revocation-and-cache.md)

---

### ORG-013 — Scope registration

| Task | Status | Depends On |
|------|--------|------------|
| ORG-013: Idempotent `RegisterScope`, `ListScopes`, `DeprecateScope`, `UndeprecateScope` | 📋 Not Started | ORG-008 |

**Scope**: Flat grammar `<service>:<action>`, `[a-z0-9_]` only, 1–50 chars, exactly one colon.
`registered_by` taken from bearer token's `client_id` — not trusted from request body.
Same-owner re-register is a no-op update; different-owner is `ErrScopeNameCollision`.
No `DeleteScope` in v1; deprecation is the only removal primitive.

See: [mvp-details/scope-model.md](mvp-details/scope-model.md)

---

### ORG-014 — Role taxonomy + membership

| Task | Status | Depends On |
|------|--------|------------|
| ORG-014: Built-in role seeding, `CreateMembership`, `UpdateMembership`, `DeleteMembership`, `CreateInvitation`, `AcceptInvitation` | 📋 Not Started | ORG-009 |

**Scope**: Four built-in roles seeded at `DefaultOrgSchema()` call: `super_admin`, `admin`,
`member`, `viewer`. Flat role model for v1; no custom-role inheritance. Invitations are
entities with expiry TTL; `AcceptInvitation` creates the `Membership` entity and expires
the `Invitation`.

See: [mvp-details/role-taxonomy.md](mvp-details/role-taxonomy.md)

---

### ORG-015 — OAuth client management

| Task | Status | Depends On |
|------|--------|------------|
| ORG-015: `CreateClient`, `GetClient`, `RotateSecret`, `AddRedirectURI`, `RemoveRedirectURI`, `DeleteClient` | 📋 Not Started | ORG-008 |

**Scope**: Client secrets are `ClientSecret` entities (Argon2id PHC hash); rotation adds a new
`ClientSecret` without removing existing ones (grace period). `DeleteClient` triggers synchronous
chunked revocation of all tokens issued to that client before soft-deleting the entity.

See: [mvp-details/data-model/oauth.md](mvp-details/data-model/oauth.md)

---

## P1: Infrastructure (depends on P0 foundation)

### ORG-016 — Cross registration + route registrar

| Task | Status | Depends On |
|------|--------|------------|
| ORG-016: All admin routes registered with Cross on startup; heartbeat per configuration.md | 📋 Not Started | ORG-005, ORG-007 |

**Scope**: `RegisterRequest` includes all routes (OAuth endpoints deliberately excluded —
handled via `/.well-known` issuer-URL contract). Heartbeat re-sends the full payload every
`ORG_REGISTRAR_INTERVAL`; Cross removes the route after missing N heartbeats.
`<ORG_ISSUER_URL>/{agencyId}/.well-known/oauth-authorization-server` served directly by Org.

See: [mvp-details/cross-registration.md](mvp-details/cross-registration.md)

---

### ORG-017 — gRPC server handlers

| Task | Status | Depends On |
|------|--------|------------|
| ORG-017: All RPCs in `internal/server/server.go` + `mappers.go` + error translation | 📋 Not Started | ORG-005, ORG-006 |

**Scope**: One handler per RPC. `mappers.go` is the sole sentinel → gRPC code / HTTP status /
OAuth error body translator. `errors.go` holds all sentinels. Internal errors produce
`codes.Internal` / `500` `server_error` with a correlation ID; real error string never crosses
the trust boundary.

See: [mvp-details/error-catalog.md](mvp-details/error-catalog.md)

---

### ORG-018 — `cmd/server/main.go` — startup wiring

| Task | Status | Depends On |
|------|--------|------------|
| ORG-018: ArangoDB connect, schema seed, Cross registrar, gRPC server wiring, graceful shutdown | 📋 Not Started | ORG-007, ORG-008, ORG-016, ORG-017 |

**Scope**: Load config, connect ArangoDB entitygraph backend, seed `DefaultOrgSchema()`
idempotently on startup, start Cross registrar heartbeat, wire gRPC onto a single TCP port.
Graceful shutdown on SIGTERM/SIGINT (30 s drain). One process per agency — `AGENCY_ID` baked
in at startup.

---

## P2: Testing

### ORG-019 — Unit tests

| Task | Status | Depends On |
|------|--------|------------|
| ORG-019: In-memory `DataManager` double, `Clock` mock, `CrossPublisher` mock | 📋 Not Started | ORG-006 |

**Scope**: Unit-test all domain logic without ArangoDB. `fakeDataManager` implements
`entitygraph.DataManager` in-memory. `fakeClock` implements `Clock`. Full negative-test
checklist per testing-strategy.md (invalid credentials, expired tokens, revoked tokens,
scope collisions, client secret grace period, reuse detection).

See: [mvp-details/testing-strategy.md](mvp-details/testing-strategy.md)

---

### ORG-020 — Integration tests

| Task | Status | Depends On |
|------|--------|------------|
| ORG-020: `testcontainers-go` ArangoDB, per-test database | 📋 Not Started | ORG-008, ORG-019 |

**Scope**: Real ArangoDB container per test run; per-test database isolation.
Tests skip without Docker. Coverage target ≥ 85% on exported functions.

See: [mvp-details/testing-strategy.md](mvp-details/testing-strategy.md)

---

### ORG-021 — Conformance tests

| Task | Status | Depends On |
|------|--------|------------|
| ORG-021: RFC 6749 + RFC 7636 + RFC 7662 + RFC 7009 + RFC 8414 flows | 📋 Not Started | ORG-018 |

**Scope**: End-to-end OAuth conformance against the running server. Gates `main` merges only.

See: [mvp-details/testing-strategy.md](mvp-details/testing-strategy.md)
