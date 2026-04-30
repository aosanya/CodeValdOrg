# Cross Registration

## Purpose

How CodeValdOrg announces itself and its routes to CodeValdCross.
Closes research-gap **Area 7 (Cross registration)**.

Builds on:

- [architecture.md §7](../../2-SoftwareDesignAndArchitecture/architecture.md) — admin route table + OAuth-endpoint table
- [configuration.md](configuration.md) — `CROSS_ENDPOINT`, `ORG_REGISTRAR_INTERVAL`

---

## Heartbeat lifecycle

`internal/registrar` runs as a goroutine started by `cmd/main.go`
after the cmux listener is up. Lifecycle:

```
Start  → wait for cmux.Serve to bind successfully
       → call OrchestratorService.Register with full RegisterRequest
       → on success, schedule next call after ORG_REGISTRAR_INTERVAL (default 20s)
       → on failure, log + retry on next tick (no backoff in v1 — fixed cadence is enough)

Loop   → every ORG_REGISTRAR_INTERVAL: Register (re-send full payload, idempotent)

Stop   → on SIGTERM / context cancel: send a final Register with status=draining
         (Cross removes the routes; in-flight requests drain via cmux graceful shutdown)
```

**Why re-send the full payload every tick** — Cross treats `Register`
as the source of truth; any route not in the most recent payload is
considered de-registered. This makes route removal a no-state-loss
operation: just stop announcing it.

**Cross-side timeout** — if Cross misses N consecutive heartbeats
(N is Cross's configuration, not Org's), it removes the routes.
Default cadence of 20s with N=3 gives ~60s of detection latency,
which is tolerable for an admin surface.

---

## `RegisterRequest` payload

The `types.ServiceRegistration` struct from `CodeValdSharedLib` carries
service identity + routes. Worked example for an Org instance serving
agency `agency-abc123`:

```go
&codevaldcrossv1.RegisterRequest{
  Service: &types.ServiceRegistration{
    ServiceName: "codevaldorg",
    InstanceID:  "<hostname>-<pid>",
    AgencyID:    "agency-abc123",          // baked-in per Q24 (one process per agency)
    Endpoint:    "<ORG_ISSUER_URL>",       // gRPC reachable address
    Status:      types.ServiceStatusActive, // → Draining on shutdown

    Routes: []types.RouteInfo{
      // Admin routes — proxied through Cross HTTP frontend
      {Method: "POST",   Path: "/{agencyId}/org",                              GRPCMethod: "InitOrganization"},
      {Method: "GET",    Path: "/{agencyId}/org",                              GRPCMethod: "GetOrganization"},
      {Method: "PATCH",  Path: "/{agencyId}/org",                              GRPCMethod: "UpdateOrganization"},
      {Method: "POST",   Path: "/{agencyId}/org/disable",                      GRPCMethod: "DisableOrganization"},
      {Method: "DELETE", Path: "/{agencyId}/org",                              GRPCMethod: "DeleteOrganization"},
      {Method: "POST",   Path: "/{agencyId}/org/users/invite",                 GRPCMethod: "InviteUser"},
      {Method: "POST",   Path: "/{agencyId}/org/invitations/accept",           GRPCMethod: "AcceptInvitation"},
      {Method: "GET",    Path: "/{agencyId}/org/users",                        GRPCMethod: "ListUsers"},
      {Method: "GET",    Path: "/{agencyId}/org/users/{userId}",               GRPCMethod: "GetUser"},
      {Method: "POST",   Path: "/{agencyId}/org/users/{userId}/suspend",       GRPCMethod: "SuspendUser"},
      {Method: "POST",   Path: "/{agencyId}/org/users/{userId}/reactivate",    GRPCMethod: "ReactivateUser"},
      {Method: "DELETE", Path: "/{agencyId}/org/users/{userId}",               GRPCMethod: "DeleteUser"},
      {Method: "POST",   Path: "/{agencyId}/org/roles",                        GRPCMethod: "CreateRole"},
      {Method: "GET",    Path: "/{agencyId}/org/roles",                        GRPCMethod: "ListRoles"},
      {Method: "PATCH",  Path: "/{agencyId}/org/roles/{roleId}",               GRPCMethod: "UpdateRole"},
      {Method: "DELETE", Path: "/{agencyId}/org/roles/{roleId}",               GRPCMethod: "DeleteRole"},
      {Method: "POST",   Path: "/{agencyId}/org/scopes",                       GRPCMethod: "RegisterScope"},
      {Method: "POST",   Path: "/{agencyId}/org/scopes/{scopeId}/deprecate",  GRPCMethod: "DeprecateScope"},
      {Method: "GET",    Path: "/{agencyId}/org/scopes",                       GRPCMethod: "ListScopes"},
      {Method: "POST",   Path: "/{agencyId}/org/memberships",                  GRPCMethod: "GrantMembership"},
      {Method: "DELETE", Path: "/{agencyId}/org/memberships/{membershipId}",   GRPCMethod: "RevokeMembership"},
      {Method: "GET",    Path: "/{agencyId}/org/memberships",                  GRPCMethod: "ListMemberships"},
      {Method: "POST",   Path: "/{agencyId}/org/oauth-clients",                GRPCMethod: "CreateOAuthClient"},
      {Method: "GET",    Path: "/{agencyId}/org/oauth-clients",                GRPCMethod: "ListOAuthClients"},
      {Method: "POST",   Path: "/{agencyId}/org/oauth-clients/{clientId}/rotate-secret",
                                                                                GRPCMethod: "RotateClientSecret"},
      {Method: "DELETE", Path: "/{agencyId}/org/oauth-clients/{clientId}",     GRPCMethod: "DeleteOAuthClient"},
      {Method: "GET",    Path: "/{agencyId}/org/audit",                        GRPCMethod: "ListAuditEvents"},
    },

    PathBindings: []types.PathBinding{
      {ParamName: "agencyId",      ParamType: types.ParamTypeAgencyID},
      {ParamName: "userId",        ParamType: types.ParamTypeEntityID},
      {ParamName: "roleId",        ParamType: types.ParamTypeEntityID},
      {ParamName: "membershipId",  ParamType: types.ParamTypeEntityID},
      {ParamName: "clientId",      ParamType: types.ParamTypeEntityID},
      {ParamName: "scopeId",       ParamType: types.ParamTypeEntityID},
    },
  },
}
```

---

## OAuth endpoints — NOT proxied through Cross

OAuth 2.0 RFC requires the `iss` claim (and the
`/.well-known/oauth-authorization-server` document) to point at the
**Authorization Server's own issuer URL**. If Cross proxied
`/oauth/authorize`, browsers would see the Cross hostname as the
issuer, breaking the OAuth metadata contract.

Therefore the five OAuth endpoints (architecture §7.2):

| Method | Path |
|---|---|
| `GET` / `POST` | `/{agencyId}/oauth/authorize` |
| `POST` | `/{agencyId}/oauth/token` |
| `POST` | `/{agencyId}/oauth/introspect` |
| `POST` | `/{agencyId}/oauth/revoke` |
| `GET` | `/{agencyId}/.well-known/oauth-authorization-server` |

are served **directly** from CodeValdOrg's HTTP listener (the cmux
HTTP/1.1 path) and are **not** included in `Routes[]`. They are
discoverable through the published metadata document (RFC 8414); Cross
does not need to know about them at the routing layer.

`ORG_ISSUER_URL` (configuration.md) is the canonical issuer string
that ends up in tokens' `iss` claims and the metadata document.

---

## Discovery — `/.well-known/oauth-authorization-server`

RFC 8414 metadata document, served at
`<ORG_ISSUER_URL>/{agencyId}/.well-known/oauth-authorization-server`:

```json
{
  "issuer":                              "<ORG_ISSUER_URL>/{agencyId}",
  "authorization_endpoint":              "<ORG_ISSUER_URL>/{agencyId}/oauth/authorize",
  "token_endpoint":                      "<ORG_ISSUER_URL>/{agencyId}/oauth/token",
  "introspection_endpoint":              "<ORG_ISSUER_URL>/{agencyId}/oauth/introspect",
  "revocation_endpoint":                 "<ORG_ISSUER_URL>/{agencyId}/oauth/revoke",
  "response_types_supported":            ["code"],
  "grant_types_supported":               ["authorization_code", "client_credentials", "refresh_token"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post", "none"],
  "code_challenge_methods_supported":    ["S256"],
  "scopes_supported":                    ["org:admin", "audit:read", "..."]   // hydrated from Scope entities
}
```

Browser clients fetch this document directly from the issuer URL —
not via Cross. Resource servers can use it to bootstrap
introspection-endpoint discovery rather than hard-coding the URL.

---

## Version negotiation

Not in v1. Backward-incompatible changes to the Cross registration
contract are coordinated by deployment ordering (Cross first, then
services). The `ServiceRegistration.ServiceName: "codevaldorg"` is the
only version-relevant token; if a future v2 changes the route layout
breakingly, a new ServiceName (`codevaldorg.v2`) is the path.
