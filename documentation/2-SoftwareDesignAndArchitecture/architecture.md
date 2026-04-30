# CodeValdOrg — Architecture

## 1. Core Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Identity protocol | [OAuth 2.0](https://oauth.net/2/) with [OAuth 2.1](https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/) tightenings | Industry standard; every modern SDK ships a client; well-documented threat model and mitigations |
| Storage granularity | One Organization per Agency | Matches CodeValdGit / CodeValdAgency per-agency isolation; zero cross-tenant queries possible from the public API |
| Storage backend | ArangoDB via `entitygraph.DataManager` (from CodeValdSharedLib) | Shared with the rest of the platform; schema declared via `DefaultOrgSchema()` and seeded idempotently |
| Transport | gRPC + HTTP on a single port via `cmux` | Consistent with CodeValdGit — gRPC for in-cluster, HTTP for OAuth endpoints and the CodeValdWorkOrg browser surface |
| Access token format (v1) | Opaque, validated only by `OrgService.Introspect` | Simple, synchronous revocation; no signing-key rotation at launch |
| Access token format (future) | Signed JWT per [RFC 9068](https://datatracker.ietf.org/doc/html/rfc9068) | Zero-round-trip validation once revocation + rotation are mature |
| PKCE | Mandatory, `S256` only | OAuth 2.1 alignment; `plain` method rejected |
| Redirect URI matching | Exact string match | Avoids open-redirect vulnerabilities |
| Refresh token policy | Rotating, single-use; ancestor chain revoked on reuse detection | Limits blast radius of stolen refresh tokens |
| Client secret storage | Argon2id hash at rest; plaintext returned once on create/rotate | Industry-standard at-rest protection |
| Admin surface | gRPC `OrgService`, exposed via CodeValdCross HTTP proxy | Zero-recompile route publishing through `registrar` heartbeat |
| Cross-service events | `CrossPublisher` interface (optional) | Publishes identity/authorization lifecycle events to CodeValdCross; `nil` = skipped (tests) |

---

## 2. Per-Agency Database Isolation

CodeValdOrg inherits the same one-agency-per-database pattern used by CodeValdGit and CodeValdAgency:

```
ArangoDB cluster
├── agency-abc123 (database)
│   ├── org_entities          ← Organization, User, Role, Membership, Scope
│   ├── org_oauth_clients     ← OAuthClient (mutable state)
│   ├── org_oauth_artifacts   ← AuthorizationCode, AccessToken, RefreshToken (TTL-indexed)
│   ├── org_audit_events      ← immutable append-only audit log
│   └── org_relationships     ← all directed graph edges
│
└── agency-xyz789 (database)
    └── … (same collection layout, fully isolated)
```

The agency ID is fixed at service-handler construction time — every `OrgService` RPC is scoped to one agency by the path parameter, and the handler selects the right database through `entitygraph.DataManager`.

---

## 3. Package Layout

```
github.com/aosanya/CodeValdOrg/
│
├── org.go                     # OrgService interface + CrossPublisher + NewOrgService
├── org_impl_org.go            # Organization lifecycle (init, get, update, disable)
├── org_impl_users.go          # User + Invitation + Membership management
├── org_impl_roles.go          # Role + Scope management
├── org_impl_oauth.go          # OAuth 2.0 endpoints (Authorize, Token, Introspect, Revoke)
├── org_impl_clients.go        # OAuth client registration, secret rotation
├── org_impl_audit.go          # Audit log append + list
├── schema.go                  # DefaultOrgSchema() — entity types seeded on startup
├── models.go                  # Domain structs (Organization, User, Role, Membership, …)
├── types.go                   # Request/response types (InviteUserRequest, TokenRequest, …)
├── errors.go                  # Sentinel errors (ErrInvalidGrant, ErrInvalidClient, …)
│
├── internal/
│   ├── server/
│   │   ├── server.go          # gRPC OrgService handler — wraps OrgService
│   │   ├── oauthhttp.go       # OAuth 2.0 HTTP handler (cmux HTTP/1.1 path)
│   │   └── mappers.go         # Proto ↔ domain model conversion
│   ├── registrar/             # Cross heartbeat — Register RPC every 20 s
│   ├── crypto/                # Argon2id hashing, PKCE S256 verification, opaque token generation
│   ├── auditlog/              # Append-only audit-event writer
│   └── config/                # Config struct + env loader
│
└── storage/
    └── arangodb/              # Thin adapter: provides collection names to entitygraph
```

---

## 4. Entity Schema

CodeValdOrg declares its entity graph through a single `DefaultOrgSchema()` function, following the same pattern as [CodeValdGit/schema.go](../../../CodeValdGit/schema.go) and [CodeValdAgency/schema.go](../../../CodeValdAgency/schema.go).

### 4.1 Core Identity Types

| Entity | Storage collection | Immutable | Purpose |
|---|---|---|---|
| `Organization` | `org_entities` | no | Root entity — one per agency; profile + enabled flag |
| `User` | `org_entities` | no | Person or service identity; status `invited` / `active` / `suspended` / `deleted` |
| `Role` | `org_entities` | no | Named bundle of scopes; built-in or custom |
| `Scope` | `org_entities` | no | Named permission string; registered by resource services at startup |
| `Membership` | `org_entities` | no | User↔Role binding within the Organization |
| `Invitation` | `org_entities` | no | Outstanding user invitation with one-time-use token and expiry |

### 4.2 OAuth 2.0 Types

| Entity | Storage collection | Immutable | Purpose |
|---|---|---|---|
| `OAuthClient` | `org_oauth_clients` | no | Registered client — `public` or `confidential`; carries redirect URIs, allowed grants, allowed scopes |
| `AuthorizationCode` | `org_oauth_artifacts` | yes (append-only, TTL ≤ 60s) | Single-use code bound to client + redirect_uri + PKCE challenge |
| `AccessToken` | `org_oauth_artifacts` | yes (TTL-indexed) | Opaque token; references its grant + user + scopes |
| `RefreshToken` | `org_oauth_artifacts` | yes (TTL-indexed) | Rotating, single-use; parent chain tracked for reuse detection |

### 4.3 Audit

| Entity | Storage collection | Immutable | Purpose |
|---|---|---|---|
| `AuditEvent` | `org_audit_events` | yes | Append-only event record with actor, subject, event_type, outcome, details |

### 4.4 Relationship Graph

```
Organization ──has_user──────────► User
             ──has_role──────────► Role
             ──has_scope─────────► Scope
             ──has_oauth_client──► OAuthClient
             ──has_audit_event───► AuditEvent (immutable)

User ──has_membership──► Membership ──grants_role──► Role
     ──has_invitation──► Invitation

Role ──has_scope──► Scope   (scopes a role bestows)

OAuthClient ──has_authorization_code──► AuthorizationCode
            ──has_access_token──────────► AccessToken
            ──has_refresh_token─────────► RefreshToken

AccessToken  ──issued_for──► User    (nullable for client-credentials)
             ──issued_to───► OAuthClient
             ──has_scope───► Scope

RefreshToken ──parent──► RefreshToken (rotation chain; self-edge only present after one rotation)
             ──issued_for──► User
             ──issued_to───► OAuthClient
```

Inverse edges (`belongs_to_organization`, `belongs_to_user`, `belongs_to_client`, …) are auto-created by `entitygraph.DataManager.CreateRelationship`.

### 4.5 Immutable Types

The following types have `Immutable: true` in the schema:
- `AuthorizationCode` — consumed once, never updated; TTL expires the record
- `AccessToken`, `RefreshToken` — opaque token records; revocation is a **separate** status collection lookup, not a mutation on the token record itself (keeps the token artifact append-only)
- `TokenRevocation` — append-only kill-record; TTL-purges when the underlying token expires naturally
- `AuditEvent` — mandated by FR-009

Revocation is modelled as a `TokenRevocation` record in `org_oauth_artifacts` referencing the token ID. `Introspect` returns `active=false` when either the token has expired or a matching revocation exists.

---

## 5. OAuth 2.0 Flow Design

See **[architecture-oauth-flows.md](architecture-oauth-flows.md)** for full sequence diagrams
and key invariants for:

- Authorization Code + PKCE (interactive clients)
- Client Credentials (service-to-service)
- Refresh Token Rotation
- Introspection
- Revocation

---

## 6. gRPC Service API — OrgService

`OrgService` is the single interface used by the gRPC server (`internal/server/server.go`). Each handler invocation is scoped to one agency via the path parameter resolved by CodeValdCross.

```go
// OrgService is the primary interface for organization and identity management.
// gRPC handlers hold this interface — never the concrete type.
// Implementations must be safe for concurrent use.
type OrgService interface {

    // ── Organization Lifecycle ──────────────────────────────────────────

    InitOrganization(ctx context.Context, req InitOrganizationRequest) (Organization, error)
    GetOrganization(ctx context.Context, agencyID string) (Organization, error)
    UpdateOrganization(ctx context.Context, req UpdateOrganizationRequest) (Organization, error)
    DisableOrganization(ctx context.Context, agencyID string) (Organization, error)
    DeleteOrganization(ctx context.Context, agencyID string) error

    // ── User & Invitation ───────────────────────────────────────────────

    InviteUser(ctx context.Context, req InviteUserRequest) (Invitation, error)
    AcceptInvitation(ctx context.Context, token string) (User, error)
    GetUser(ctx context.Context, userID string) (User, error)
    ListUsers(ctx context.Context, filter UserFilter) ([]User, error)
    SuspendUser(ctx context.Context, userID string) (User, error)
    ReactivateUser(ctx context.Context, userID string) (User, error)
    DeleteUser(ctx context.Context, userID string) error

    // ── Roles & Scopes ──────────────────────────────────────────────────

    CreateRole(ctx context.Context, req CreateRoleRequest) (Role, error)
    UpdateRole(ctx context.Context, req UpdateRoleRequest) (Role, error)
    DeleteRole(ctx context.Context, roleID string) error
    ListRoles(ctx context.Context) ([]Role, error)

    RegisterScope(ctx context.Context, req RegisterScopeRequest) (Scope, error)
    DeprecateScope(ctx context.Context, scopeID string) (Scope, error)
    ListScopes(ctx context.Context) ([]Scope, error)

    // ── Membership ──────────────────────────────────────────────────────

    GrantMembership(ctx context.Context, req GrantMembershipRequest) (Membership, error)
    RevokeMembership(ctx context.Context, membershipID string) (Membership, error)
    ListMemberships(ctx context.Context, filter MembershipFilter) ([]Membership, error)

    // ── OAuth Clients ───────────────────────────────────────────────────

    CreateOAuthClient(ctx context.Context, req CreateOAuthClientRequest) (OAuthClient, string, error) // (client, plaintext_secret)
    RotateClientSecret(ctx context.Context, clientID string) (string, error)                          // new plaintext_secret
    ListOAuthClients(ctx context.Context) ([]OAuthClient, error)
    DeleteOAuthClient(ctx context.Context, clientID string) error

    // ── OAuth 2.0 Protocol Endpoints ────────────────────────────────────

    Authorize(ctx context.Context, req AuthorizeRequest) (AuthorizeResponse, error) // issues AuthorizationCode
    Token(ctx context.Context, req TokenRequest) (TokenResponse, error)             // grant_type dispatch
    Introspect(ctx context.Context, req IntrospectRequest) (IntrospectResponse, error)
    Revoke(ctx context.Context, req RevokeRequest) error

    // ── Audit ───────────────────────────────────────────────────────────

    ListAuditEvents(ctx context.Context, filter AuditFilter) ([]AuditEvent, error)
}

// NewOrgService constructs an OrgService backed by the given DataManager and SchemaManager.
// agencyID scopes the instance to a single agency's database.
// pub may be nil — cross-service events are skipped when no publisher is set.
func NewOrgService(
    dm entitygraph.DataManager,
    sm OrgSchemaManager,
    pub CrossPublisher,
    agencyID string,
) OrgService
```

---

## 7. HTTP Surface (via CodeValdCross)

All admin routes are declared in the `RegisterRequest.routes` payload sent to Cross. The OAuth 2.0 endpoints are served directly on the HTTP side of the `cmux` multiplexer (they must be reachable by browsers and external clients, not just through Cross).

### 7.1 Admin Routes (through Cross HTTP proxy)

| HTTP Method | HTTP Path | gRPC Method |
|---|---|---|
| `POST` | `/{agencyId}/org` | `InitOrganization` |
| `GET` | `/{agencyId}/org` | `GetOrganization` |
| `PATCH` | `/{agencyId}/org` | `UpdateOrganization` |
| `POST` | `/{agencyId}/org/disable` | `DisableOrganization` |
| `DELETE` | `/{agencyId}/org` | `DeleteOrganization` |
| `POST` | `/{agencyId}/org/users/invite` | `InviteUser` |
| `POST` | `/{agencyId}/org/invitations/accept` | `AcceptInvitation` |
| `GET` | `/{agencyId}/org/users` | `ListUsers` |
| `GET` | `/{agencyId}/org/users/{userId}` | `GetUser` |
| `POST` | `/{agencyId}/org/users/{userId}/suspend` | `SuspendUser` |
| `POST` | `/{agencyId}/org/users/{userId}/reactivate` | `ReactivateUser` |
| `DELETE` | `/{agencyId}/org/users/{userId}` | `DeleteUser` |
| `POST` | `/{agencyId}/org/roles` | `CreateRole` |
| `GET` | `/{agencyId}/org/roles` | `ListRoles` |
| `PATCH` | `/{agencyId}/org/roles/{roleId}` | `UpdateRole` |
| `DELETE` | `/{agencyId}/org/roles/{roleId}` | `DeleteRole` |
| `POST` | `/{agencyId}/org/scopes` | `RegisterScope` |
| `POST` | `/{agencyId}/org/scopes/{scopeId}/deprecate` | `DeprecateScope` |
| `GET` | `/{agencyId}/org/scopes` | `ListScopes` |
| `POST` | `/{agencyId}/org/memberships` | `GrantMembership` |
| `DELETE` | `/{agencyId}/org/memberships/{membershipId}` | `RevokeMembership` |
| `GET` | `/{agencyId}/org/memberships` | `ListMemberships` |
| `POST` | `/{agencyId}/org/oauth-clients` | `CreateOAuthClient` |
| `GET` | `/{agencyId}/org/oauth-clients` | `ListOAuthClients` |
| `POST` | `/{agencyId}/org/oauth-clients/{clientId}/rotate-secret` | `RotateClientSecret` |
| `DELETE` | `/{agencyId}/org/oauth-clients/{clientId}` | `DeleteOAuthClient` |
| `GET` | `/{agencyId}/org/audit` | `ListAuditEvents` |

### 7.2 OAuth 2.0 Endpoints (direct HTTP, RFC-compliant)

| Method | Path | Purpose |
|---|---|---|
| `GET` / `POST` | `/{agencyId}/oauth/authorize` | Authorization Code request — returns 302 with `?code=…&state=…` |
| `POST` | `/{agencyId}/oauth/token` | Token endpoint — dispatches on `grant_type` |
| `POST` | `/{agencyId}/oauth/introspect` | RFC 7662 introspection |
| `POST` | `/{agencyId}/oauth/revoke` | RFC 7009 revocation |
| `GET` | `/{agencyId}/.well-known/oauth-authorization-server` | Server metadata (RFC 8414) |

These endpoints **do not** pass through the Cross proxy — they are served directly from CodeValdOrg's HTTP listener (cmux HTTP/1.1 path), because OAuth clients expect the issuer URL to be the direct service address.

### 7.3 Pub/Sub Events

After each successful mutating operation, CodeValdOrg publishes a typed event via its `CrossPublisher`:

| Event | Topic | Trigger |
|---|---|---|
| Organization created | `cross.org.{agencyID}.organization.created` | `InitOrganization` |
| User invited | `cross.org.{agencyID}.user.invited` | `InviteUser` |
| User activated | `cross.org.{agencyID}.user.activated` | `AcceptInvitation` |
| User suspended | `cross.org.{agencyID}.user.suspended` | `SuspendUser` |
| Membership granted | `cross.org.{agencyID}.membership.granted` | `GrantMembership` |
| Token issued | `cross.org.{agencyID}.token.issued` | `Token` (all grant types) |
| Token revoked | `cross.org.{agencyID}.token.revoked` | `Revoke` + refresh rotation reuse |

---

## 8. CodeValdSharedLib Dependency

CodeValdOrg imports `github.com/aosanya/CodeValdSharedLib` for:

| SharedLib package | What CodeValdOrg uses it for |
|---|---|
| `entitygraph` | `DataManager` and `SchemaManager` — backs every identity and OAuth entity in the per-agency database |
| `registrar` | Generic Cross heartbeat — sends `Register` RPC every 20 s; routes declared once |
| `serverutil` | `NewGRPCServer` (enables gRPC reflection for the Cross proxy), `RunWithGracefulShutdown`, env helpers |
| `arangoutil` | `Connect(ctx, Config)` — ArangoDB bootstrap in `storage/arangodb` |
| `gen/go/codevaldcross/v1` | Generated stubs for the Cross `OrchestratorService` (registrar heartbeat) |
| `types` | `PathBinding`, `RouteInfo`, `ServiceRegistration` — shared with Cross when declaring routes |

> **Principle**: Infrastructure code shared across services lives in SharedLib. CodeValdOrg retains only its identity/OAuth domain logic (`OrgService`), domain errors, gRPC + HTTP handlers, and storage schema (`schema.go`).

---

## 9. Security Design Summary

| Concern | Mitigation |
|---|---|
| Stolen access token | Opaque tokens + synchronous introspection — revocation takes effect immediately |
| Stolen refresh token | Rotating single-use refresh tokens; reuse revokes the whole chain |
| Open redirect via `redirect_uri` | Exact-match allow-list; wildcard matching rejected |
| PKCE downgrade | `plain` method rejected; `S256` required |
| Client secret leakage at rest | Argon2id hash; plaintext returned once on create/rotate |
| Replay of authorization code | Single-use, ≤ 60 s TTL, bound to `client_id` + `redirect_uri` + PKCE challenge |
| Cross-agency access | Per-agency database; handlers scoped by path parameter; no multi-agency query path exists |
| Audit tampering | `AuditEvent` is `Immutable: true` in the schema — `UpdateEntity` returns `ErrImmutableType` |
| Token forgery | Opaque tokens are server-generated; no client-side signature to forge |

---

## 10. Open Architectural Topics

Three topics are called out for future detailed design documents (to follow the CodeValdGit pattern of per-topic architecture files):

| Topic | Future file | Why separate |
|---|---|---|
| Signed JWT access tokens (RFC 9068) with JWKS rotation | `architecture-jwt-tokens.md` | Non-trivial — revocation story + signing-key rotation require their own design pass |
| External identity provider federation (OIDC, SAML, social) | `architecture-federation.md` | Adds a whole new subsystem; independent of the core OAuth 2.0 server |
| High-volume introspection caching for in-cluster resource servers | `architecture-introspection-cache.md` | Performance concern; only relevant once NFR-002 latency budget is at risk |
