# Problem Definition

## The Problem

The CodeVald platform runs agencies that coordinate AI agents, produce artifacts, and expose user-facing surfaces (CodeValdHi, CodeValdWorkFrontend, CodeValdGitFrontend). Until now, the platform has had **no dedicated identity or access-management service**:

| Today's gap | Consequence |
|---|---|
| No canonical User record per agency | Services invent their own "user" concept; identities diverge and drift |
| No OAuth 2.0 endpoints | Every client has to invent a bespoke auth handshake; third-party integrations are impossible |
| No role / scope vocabulary | Authorization checks are scattered across services, often implicit in code |
| No audit trail for identity events | Who invited whom, when a token was revoked, which client was rotated — untraceable |
| No admin surface | Platform operators have no safe, consistent way to manage users per agency |

Without a central identity service, every new surface (frontend, CLI, integration) would re-implement authentication, and every resource service would enforce access control differently. The result would be fragile, inconsistent, and impossible to audit.

---

## The Solution

**CodeValdOrg** — a Go gRPC microservice that provides:

- **Per-agency Organization** — one Organization per Agency, isolated in its own database (matches CodeValdGit / CodeValdAgency)
- **User, Role, and Membership management** — a single canonical identity graph per agency
- **Standards-based OAuth 2.0** — Authorization Code with PKCE for interactive clients; Client Credentials for services; introspection and revocation for resource servers
- **Scope vocabulary** — resource services register their scopes at startup; roles own scope sets; memberships grant their holders the union of their roles' scopes
- **Immutable audit log** — every mutation is recorded
- **Admin management surface** — consumed by the CodeValdWorkOrg frontend through the CodeValdCross HTTP proxy

---

## Why OAuth 2.0

[OAuth 2.0](https://oauth.net/2/) is the industry-standard authorization protocol. Choosing it means:

| Benefit | Reason |
|---|---|
| Zero-custom client integration | Every modern SDK (Go, TypeScript, Swift, Kotlin) has a ready OAuth 2.0 client |
| Native third-party integration path | External apps can request authorization without a CodeVald-specific handshake |
| Well-understood threat model | PKCE, redirect-URI exact-match, token revocation are all documented mitigations |
| Resource-server ergonomics | Any service inside the platform can check `Authorization: Bearer …` via a single introspection call |
| Future-proof | JWT access tokens, DPoP, RFC 9068 profile can be layered in later without re-building the foundation |

V1 adopts the **OAuth 2.1** tightenings that matter most: mandatory PKCE, `S256` only, exact-match redirect URIs, and no implicit flow.

---

## Scope of Ownership

CodeValdOrg owns the **identity and authorization** concern. It does **not** own:

| Not owned by CodeValdOrg | Where it lives |
|---|---|
| Agency mission, goals, workflows | CodeValdAgency |
| Artifact storage and versioning | CodeValdGit |
| Work orchestration | CodeValdWork |
| Cross-service routing and proxying | CodeValdCross |
| AI agent execution | CodeValdAI |

CodeValdOrg's output is a **signed identity context** (bearer token + introspection response) that every other service trusts.
