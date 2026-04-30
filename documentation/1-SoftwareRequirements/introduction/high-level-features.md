# High-Level Features

## Feature Overview

CodeValdOrg provides the following top-level capabilities:

---

### 1. Organization Management
- **One Organization per Agency** — created automatically at Agency creation
- **Profile** — name, slug, description, support contact, branding metadata
- **Update** — admin can edit profile fields
- **Disable / enable** — soft-switch for compliance holds; all live tokens revoked on disable

### 2. User Management
- **Invite** — admin issues an invitation email; User status = `invited`
- **Activate** — User becomes `active` on first successful OAuth authorization
- **Suspend** — all live tokens revoked; future flows denied; record retained for audit
- **Soft-delete** — tombstones the User; explicit `PurgeUser` required for permanent removal
- **Lookup** — by `user_id`, `email`, or external identity provider subject

### 3. Role and Scope Management
- **Built-in roles** — `super_admin`, `admin`, `member`, `viewer` — always present, cannot be deleted or renamed
- **Custom roles** — admin creates role with agency-specific name and scope set
- **Scope registration** — resource services register their scopes at service-start time
- **Scope resolution** — given a User and requested scopes, the service returns the effective granted scopes

### 4. Membership Management
- **Grant** — assign a User one or more Roles
- **Revoke** — remove a Role from a User; token revocation cascades if the User loses all memberships
- **List** — paginated listing of all Users with their roles for admin UIs

### 5. OAuth 2.0 Authorization Server
- **Authorization Code + PKCE** (RFC 6749 §4.1 + RFC 7636) — interactive browser / mobile flow
- **Client Credentials** (RFC 6749 §4.4) — service-to-service flow
- **Refresh Token** (RFC 6749 §6) — rotating refresh tokens
- **Introspection** (RFC 7662) — authoritative token validity check over HTTP and gRPC
- **Revocation** (RFC 7009) — immediate invalidation of access or refresh tokens

### 6. OAuth Client Registration
- **Public clients** — SPA, mobile; no `client_secret`; PKCE mandatory
- **Confidential clients** — server-side apps; `client_secret` hashed at rest (Argon2id)
- **Secret rotation** — returns new plaintext once; old secret accepted for a short grace window
- **Redirect URI allow-list** — exact-match; wildcard matching refused

### 7. Audit Log
- **Append-only** — every identity/authorization mutation recorded with actor, subject, outcome
- **Filterable read** — admin UI can query by actor, subject, event type, time range
- **Retention policy** — configurable per deployment; default 365 days

### 8. Admin Management Surface (CodeValdWorkOrg)
- All admin operations exposed as gRPC methods on `OrgService`
- Routed through CodeValdCross HTTP proxy for browser consumption
- Every call requires a bearer token carrying the `org:admin` scope

---

## What CodeValdOrg Does NOT Do

| Out of Scope | Reason |
|---|---|
| SAML / OIDC federation | Planned for v2 — not required at launch |
| Social login (Google, GitHub, …) | Federation adapter can be added later without re-building core |
| End-user password self-service UI | The UI layer handles this; CodeValdOrg exposes only the underlying endpoints |
| Cross-agency queries or sharing | Agencies are isolated at the database level; by design |
| Multi-factor / biometric enrolment | Layered on top in v2; v1 is password + invitation |
| Billing, usage metering | Separate concern, lives elsewhere in the platform |
