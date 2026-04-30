# CodeValdOrg — Requirements

## 1. Purpose

CodeValdOrg is a **Go gRPC microservice** that provides organizational identity and access management for the CodeVald platform.

Every Agency is owned by an **Organization**. The Organization is the administrative boundary that holds users, roles, memberships, and OAuth 2.0 clients/tokens. CodeValdOrg is the authoritative source of truth for "who is acting, what are they allowed to do, and did they prove it with a valid credential?"

Identity and authorization follow [OAuth 2.0](https://oauth.net/2/) so that any standards-compliant client (browser SPA, mobile app, CLI, service) can authenticate without CodeVald-specific protocols.

---

## 2. Scope

### In Scope
- Organization profile — name, slug, contact, branding metadata
- User lifecycle — invite, activate, suspend, soft-delete
- Role lifecycle — built-in (`super_admin`, `admin`, `member`, `viewer`) + agency-defined custom roles
- Membership — bind a User to one or more Roles within the Organization
- OAuth 2.0 Authorization Server:
  - Authorization Code flow **with PKCE** (RFC 6749 §4.1 + RFC 7636) for interactive clients
  - Client Credentials flow (RFC 6749 §4.4) for service-to-service
  - Refresh Token flow (RFC 6749 §6)
  - Token introspection (RFC 7662)
  - Token revocation (RFC 7009)
- OAuth Clients — public (SPA/mobile) and confidential (service) clients, per-org registration
- Scopes — named permission units; resolvable by the policy layer
- Audit log — every identity/authorization event persisted with actor, subject, outcome
- Admin management surface — exposed to CodeValdWorkOrg through CodeValdCross HTTP proxy

### Out of Scope
- SAML / OIDC Provider federation (v1 — may be added later as an identity source)
- End-user self-service password management (v1 treats password auth as an external identity source)
- Social-login (Google / GitHub / etc.) federation in v1
- Multi-region replication or active-active failover
- Biometric / WebAuthn second factors in v1

---

## 3. Functional Requirements

### FR-001: One Organization Per Agency

- Each Agency owns **exactly one Organization** entity, stored in the Agency's own database
- The Agency ID is the partition key — CodeValdOrg follows the same per-agency database isolation pattern as CodeValdGit and CodeValdAgency
- The Organization is created automatically when the Agency is created; it cannot be deleted independently of the Agency
- Organization metadata (name, slug, description, contact) is mutable by an `admin` role actor

### FR-002: User Lifecycle

- Create a User record by email — status `invited` until the user accepts
- Activate a User on first successful OAuth authorization
- Suspend a User — all live tokens revoked, future flows denied, record retained for audit
- Soft-delete a User — suspends + tombstones; hard delete only via explicit operator purge
- Users are scoped to the Organization — the same email in two Organizations is two distinct User records

### FR-003: Role Lifecycle

- Built-in roles always present: `super_admin`, `admin`, `member`, `viewer`
- Built-in roles **cannot be deleted or renamed** but their scope bindings can be extended
- Admins may create **custom roles** with agency-specific names and scope sets
- Custom roles are mutable (rename, re-scope, delete) as long as no active membership depends on an undefined state

### FR-004: Membership

- A Membership binds a `User` to one or more `Role` entities within the Organization
- A User with zero active memberships cannot obtain a token (flow fails with `access_denied`)
- Memberships are directly readable by the policy layer via gRPC so that CodeValdCross can resolve effective scopes at token mint time

### FR-005: OAuth 2.0 Authorization Code Flow with PKCE

- CodeValdOrg implements the **Authorization Code grant with PKCE** (RFC 7636)
- PKCE is **required** for all public clients and **recommended** for confidential clients
- Authorization endpoint validates `code_challenge` + `code_challenge_method` (`S256` only; `plain` rejected in v1)
- Authorization codes are single-use, bound to the `client_id`, `redirect_uri`, and PKCE verifier
- Authorization codes expire in **≤ 60 seconds**
- Redirect URIs must match a pre-registered value exactly (no wildcard matching)

### FR-006: OAuth 2.0 Client Credentials Flow

- Confidential clients may obtain tokens via the **Client Credentials grant** (RFC 6749 §4.4)
- Used for service-to-service calls (e.g. CodeValdAI → CodeValdCross)
- Client authentication is `client_secret_basic` or `client_secret_post` (no `none` for client-credentials)
- Scopes on the issued token are the **intersection** of the client's registered scopes and the requested scopes

### FR-007: Token Introspection and Revocation

- **Introspection** (RFC 7662): `POST /oauth/introspect` — returns `active`, `scope`, `sub`, `client_id`, `exp`, `iat`
- Introspection is the authoritative check at resource-server call time; tokens are **opaque** in v1 (JWT access tokens may be added in v2)
- **Revocation** (RFC 7009): `POST /oauth/revoke` — immediately invalidates an access or refresh token
- Introspection must be callable over gRPC as well as HTTP for in-cluster policy checks

### FR-008: Scope-Based Resource Permissions

- A Scope is a named permission string (e.g. `agency:read`, `git:write`, `work:admin`)
- Scopes are declared by the resource services (CodeValdGit, CodeValdWork, etc.) and registered with CodeValdOrg at service start-up
- Roles own a set of scopes; Memberships grant their User the union of their roles' scopes
- At token mint time, the effective scope is **min(requested, user's grant, client's registered)**

### FR-009: Audit Log

- Every mutating identity or authorization event is appended to an immutable audit log:
  - User invited, activated, suspended, deleted
  - Role created, updated, deleted; scope added / removed
  - Membership granted / revoked
  - OAuth client created / rotated / revoked
  - Authorization code issued / consumed / expired
  - Token issued / refreshed / revoked / introspected-denied
- Entries include `actor_id`, `subject_id`, `event_type`, `event_at`, `source_ip`, `outcome`, structured `details`
- The audit log is append-only — no UpdateEntity path exists for audit records

### FR-010: Admin Management Surface

- All admin operations (Organization metadata, user / role / membership / client CRUD, audit log read) are exposed as gRPC methods on `OrgService`
- The CodeValdWorkOrg frontend consumes these through the CodeValdCross HTTP proxy (no direct gRPC from the browser)
- Every admin operation requires a token with the `org:admin` scope or higher

---

## 4. Non-Functional Requirements

### NFR-001: Standards Compliance
- OAuth 2.0 endpoints must pass a standard conformance suite (e.g. OIDF certification fixtures re-used where licensing permits)
- Error responses follow RFC 6749 `error` / `error_description` body shape

### NFR-002: Low-Latency Introspection
- p99 introspection latency **≤ 10 ms** in-cluster over gRPC
- Introspection must not block on any external call — all data is resolvable from the local agency database

### NFR-003: Per-Agency Isolation
- No cross-agency queries are possible from the public API — every handler is scoped by `agencyId` at entry
- Agency databases are fully independent; deleting an Agency's database removes its entire identity graph

### NFR-004: Secure Secret Handling
- `client_secret` is hashed at rest (Argon2id); plaintext returned **once only** on client creation / rotation
- Refresh tokens are hashed at rest
- PKCE verifiers are never persisted — only the challenge is stored, discarded at code consumption

### NFR-005: Schema-Driven
- CodeValdOrg declares its entity graph through a single `DefaultOrgSchema()` function (matches CodeValdGit / CodeValdAgency pattern)
- Schema is seeded idempotently on startup through `entitygraph.SchemaManager`

### NFR-006: No External Identity Provider Dependency (v1)
- The service must be usable with password + invitation only
- Identity provider federation (OIDC, SAML, social login) is an additive feature, not a dependency

---

## 5. Open Questions

| # | Question | Impact |
|---|---|---|
| OQ-001 | Should access tokens be opaque (introspection only) or JWT-signed? v1 is opaque to keep revocation simple; v2 may add signed JWTs for zero-round-trip checks | Token format, introspection caching strategy |
| OQ-002 | Should refresh-token rotation be mandatory (RFC 6749 §10.4)? Leaning yes — rotating refresh tokens defangs stolen-token replay | Client complexity increases slightly |
| OQ-003 | Does first-party auth (password + email) belong in CodeValdOrg, or should it always delegate to an external IdP? v1 supports both; the decision affects long-term surface area | Complexity, support burden |
| OQ-004 | Which OAuth 2.1 tightenings do we adopt for v1? (Mandatory PKCE, no `plain` method, exact-match redirect URI — already planned; implicit flow disallowed — already planned) | Drift from 2.0 vs. simpler client integration |
| OQ-005 | Should audit events flow through CodeValdCross pub/sub as well as local storage so a central SIEM can consume them? | Operational observability vs. coupling |
