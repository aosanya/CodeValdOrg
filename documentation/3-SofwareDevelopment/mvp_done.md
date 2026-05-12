# MVP Done — Completed Tasks

Completed tasks are removed from `mvp.md` and recorded here with their completion date.

| Task ID | Title | Completion Date | Branch | Notes |
|---------|-------|-----------------|--------|-------|
| ORG-001 | Module Scaffolding | 2026-05-11 | main | `go.mod`, `buf.yaml`, `buf.gen.yaml`, `Makefile`, `Dockerfile.server`, stub `cmd/server/main.go` |
| ORG-002 | `schema.go` — `DefaultOrgSchema()` | 2026-05-11 | main | 15 TypeDefinitions across 4 storage collections; immutability flags on oauth artifacts and audit events |
| ORG-003 | `models.go` — Go value types | 2026-05-11 | main | All domain structs, enums, request/response types, `Clock` interface |
| ORG-004 | `errors.go` — sentinel errors | 2026-05-11 | main | Full error catalog; `mappers.go` sole sentinel→gRPC code translation point |
| ORG-005 | Proto + codegen | 2026-05-11 | main | `proto/codevaldorg/v1/org.proto` with all RPCs; `buf generate` → `gen/go/codevaldorg/v1/`; added `user_id` to `AuthorizeRequest` |
| ORG-006 | `OrgManager` interface | 2026-05-11 | main | Flat interface + `orgManager` struct holding `dm`, `sm`, `pub`, `clock`, `cfg` |
| ORG-007 | `internal/config/config.go` | 2026-05-11 | main | Two-pass validation; 6 required + 12 optional env vars |
| ORG-008 | ArangoDB entitygraph backend | 2026-05-11 | main | Thin adapter over `SharedLib entitygraph/arangodb`; fixed collection names matching spec |
| ORG-009 | Organization + User lifecycle | 2026-05-11 | main | Full CRUD + `InviteUser`, `AcceptInvitation`, `SuspendUser`, `ReactivateUser`, built-in role seeding |
| ORG-010 | Token issuance | 2026-05-11 | main | Auth-code+PKCE, client-credentials, refresh-token rotation; strong-CP publish-or-rollback; HTTP handlers wired |
| ORG-011 | Token introspection | 2026-05-11 | main | Hash-index lookup; `{active:false}` parity; scope/sub/client_id resolution; HTTP introspect endpoint |
| ORG-012 | Token revocation + pub/sub | 2026-05-11 | main | Revoke by hash; publish `org.token.revoked`; HTTP revoke endpoint |
| ORG-013 | Scope registration | 2026-05-11 | main | Idempotent `RegisterScope`, `DeprecateScope`, `ListScopes`; flat grammar validation; reserved-prefix guard |
| ORG-014 | Role taxonomy + membership | 2026-05-11 | main | 4 built-in roles seeded at init; `GrantMembership`, `RevokeMembership`, `ListMemberships`, `InviteUser`, `AcceptInvitation` |
| ORG-015 | OAuth client management | 2026-05-11 | main | `CreateClient`, `RotateSecret`, `ListClients`, `DeleteClient`; Argon2id PHC secrets; grace-period rotation |
| ORG-016 | Cross registration + route registrar | 2026-05-11 | main | `internal/registrar`; heartbeat every `ORG_REGISTRAR_INTERVAL`; `/.well-known/oauth-authorization-server` served directly |
| ORG-017 | gRPC server handlers | 2026-05-11 | main | All RPCs in `internal/server/server.go`; `mappers.go` sole error translator |
| ORG-018 | `cmd/server/main.go` — startup wiring | 2026-05-11 | main | ArangoDB connect, schema seed, Cross registrar, cmux gRPC+HTTP, 30 s graceful shutdown |
