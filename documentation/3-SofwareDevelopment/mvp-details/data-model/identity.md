# Identity Entities

Seven entities form the identity graph: `Organization`, `User`,
`PasswordCredential`, `Role`, `Scope`, `Membership`, `Invitation`. All
live in the `org_entities` storage collection.

---

## `Organization`

Root entity — exactly one per agency (FR-001). `disabled_at` and
`deleted_at` are distinct lifecycle states.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition + unique |
| Core | `name` | string | yes | — |
| Core | `enabled` | bool | yes | filter index |
| Long tail | `description` | string | no | — |
| Long tail | `contact_email` | string | no | — |
| Long tail | `logo_url` | string | no | — |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `updated_at` | string (RFC 3339) | no | — |
| Long tail | `disabled_at` | string (RFC 3339, nullable) | no | — |
| Long tail | `deleted_at` | string (RFC 3339, nullable) | no | — |

**Edges out:** `has_user`, `has_role`, `has_scope`, `has_oauth_client`,
`has_audit_event`.

---

## `User`

A person or service identity within the Organization. Credentials are
not stored on `User` — they live on a separate `PasswordCredential`
(and, in v2, `WebAuthnCredential` / `OIDCCredential`) bound by edge.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `email` | string | yes | unique `(agency_id, email)` |
| Core | `status` | option (`invited` / `active` / `suspended` / `deleted`) | yes | filter index |
| Long tail | `display_name` | string | no | — |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `updated_at` | string (RFC 3339) | no | — |
| Long tail | `deleted_at` | string (RFC 3339, nullable) | no | — |

**Edges out:** `has_password_credential`, `has_membership`,
`has_invitation`. **Edges in:** `belongs_to_organization`
(auto-inverse of `Organization.has_user`).

`last_login_at` is **not** a stored field — it is a derived view of
`PasswordCredential.last_used_at` (or, in v2, the most-recent
`last_used_at` across any credential bound to the user).

---

## `PasswordCredential`

v1 ships only the password kind. v2 will add `WebAuthnCredential` and
`OIDCCredential` as **separate entity types** (kind-per-type model
chosen in Q6 of the research Q&A).

The Argon2id PHC string carries algorithm + params + salt inline, so
no separate fields are needed for those.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `password_hash` | string (Argon2id PHC) | yes | — |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `last_used_at` | string (RFC 3339) | no | — |
| Long tail | `revoked_at` | string (RFC 3339, nullable) | no | — |
| Long tail | `expires_at` | string (RFC 3339, nullable) | no | — |

**Edges in:** `belongs_to_user` (auto-inverse of
`User.has_password_credential`).

Lookup at `/token`: `email → User → has_password_credential →
PasswordCredential`. Filter to the active credential by
`revoked_at == null`. Single-active-credential invariant is enforced at
service layer, not by the schema.

---

## `Role`

Built-in (`super_admin`, `admin`, `member`, `viewer`) and custom roles
share this type, distinguished by the `builtin` flag (FR-003 — built-ins
cannot be deleted or renamed).

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `name` | string | yes | unique `(agency_id, name)` |
| Core | `builtin` | bool | yes | filter index |
| Long tail | `display_name` | string | no | — |
| Long tail | `description` | string | no | — |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `updated_at` | string (RFC 3339) | no | — |
| Long tail | `deleted_at` | string (RFC 3339, nullable) | no | — |

**Edges out:** `has_scope` (one per scope this role grants).

There is no role-hierarchy field. The four built-ins'
"super_admin > admin > member > viewer" ordering is implicit via scope
membership. Custom roles enumerate their full scope set as direct
`has_scope` edges — no `parent_role` edge in v1 (BR-004 resolved 2026-04-27,
see [../role-taxonomy.md](../role-taxonomy.md)). Inheritance can be
added in v2 without breaking existing data because resource servers
only ever see the flat effective-scope set on introspected tokens.

---

## `Scope`

The agency-scoped registry of named permission strings. Resource
services register their scopes at startup (FR-008).

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `name` | string (e.g. `git:read`) | yes | unique `(agency_id, name)` |
| Core | `registered_by` | string (service name, e.g. `codevaldgit`) | yes | filter index |
| Long tail | `description` | string | no | — |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `updated_at` | string (RFC 3339) | no | — |
| Long tail | `deprecated_at` | string (RFC 3339, nullable) | no | — |

**Edges in:** `belongs_to_organization` (auto-inverse of
`Organization.has_scope`); incoming `has_scope` from `Role`;
`allows_scope` from `OAuthClient`; `has_scope` from `AccessToken` /
`RefreshToken`; `has_requested_scope` from `AuthorizationCode`.

Scope-name grammar (allowed characters, wildcards, hierarchy) is the
job of research-gap Area 4 — not yet specified.

---

## `Membership`

Binds a `User` to a `Role` (FR-004). Pure metadata; the actual link
lives in the two edges.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Long tail | `granted_at` | string (RFC 3339) | no | — |
| Long tail | `granted_by` | string (user_id of admin) | no | — |
| Long tail | `revoked_at` | string (RFC 3339, nullable) | no | — |
| Long tail | `revoked_by` | string (user_id, nullable) | no | — |

**Edges in:** `belongs_to_user` (auto-inverse of
`User.has_membership`). **Edges out:** `grants_role`.

Duplicate-prevention for active `(User, Role)` pairs is service-layer:
`GrantMembership` checks for an existing active membership and is
idempotent on re-grant. Revocation keeps the entity (sets
`revoked_at`) so audit reads can reach it without resurrection.

---

## `Invitation`

Outstanding user invitation with a one-time-use token (FR-002). The
`User` record is created in status `invited` simultaneously with the
`Invitation`; on acceptance, `User.status` flips to `active` and
`Invitation.status` flips to `accepted`.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `token_hash` | string (SHA-256, base64url) | yes | unique `(agency_id, token_hash)` |
| Core | `status` | option (`pending` / `accepted` / `expired` / `revoked`) | yes | filter index |
| Core | `expires_at` | string (RFC 3339) | yes | — |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `accepted_at` | string (RFC 3339, nullable) | no | — |
| Long tail | `revoked_at` | string (RFC 3339, nullable) | no | — |

**Edges in:** `belongs_to_user`. **Edges out:** `invited_by` (→ User
who issued), `will_grant_role` (one per intended role; resolved into
`Membership` entities at acceptance).

Plaintext token is in the email link only. Lookup at acceptance is by
hash equality — no constant-time compare needed. SHA-256 (not
Argon2id) is correct here because tokens are high-entropy random
bytes; there is nothing to brute-force.
