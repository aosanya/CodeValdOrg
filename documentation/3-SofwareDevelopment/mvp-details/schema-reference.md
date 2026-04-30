# Schema Reference

## Purpose

The complete index → collection → immutability → entity-type matrix
for `DefaultOrgSchema()`. Companion to (eventually) `schema.go`.
Closes research-gap **Area 8 (ArangoDB schema)** at the documentation
level — the corresponding code change is to write `schema.go` itself.

The per-entity property tables live in
[data-model/](data-model/); this doc focuses on what cuts across them
(collection routing, immutability, indexes, edge inventory).

---

## Storage collections

Three document collections + one edge collection per agency database:

| Collection | Stores | Immutable? | Notes |
|---|---|---|---|
| `org_entities` | Identity entities: Organization, User, PasswordCredential, Role, Scope, Membership, Invitation | mixed (per-type flag) | Identity domain |
| `org_oauth_clients` | OAuth client metadata: OAuthClient, ClientSecret, RedirectURI | no | Mutable client registry |
| `org_oauth_artifacts` | OAuth artifacts: AuthorizationCode, AccessToken, RefreshToken, TokenRevocation | yes (all types) | TTL-indexed on `expires_at` |
| `org_audit_events` | AuditEvent | yes | Append-only forensic log |
| `org_relationships` | All edges (`has_*`, `belongs_to_*`, `parent`, `issued_to`, `issued_for`, `has_scope`, `grants_role`, `will_grant_role`, `allows_scope`, `invited_by`) | n/a (edges) | One edge collection per agency |

---

## Indexes

The minimum set required by the v1 query patterns:

### `org_entities`

| Index | Type | Purpose |
|---|---|---|
| `(typeID, properties.agency_id, properties.email)` | unique | Lookup `User` by email at resource-owner path; uniqueness within agency |
| `(typeID, properties.agency_id)` where typeID = "Organization" | unique | One Organization per agency (FR-001) |
| `(typeID, properties.agency_id, properties.name)` | unique | Lookup `Role`, `Scope` by name; uniqueness within agency |
| `(typeID, properties.agency_id, properties.token_hash)` | unique | Lookup `Invitation` at `AcceptInvitation` |
| `(typeID, properties.agency_id, properties.status)` | non-unique | Filter `User` by status for admin lists |
| `(typeID, properties.agency_id, properties.builtin)` | non-unique | Filter `Role` by built-in vs custom |

### `org_oauth_clients`

| Index | Type | Purpose |
|---|---|---|
| `(typeID, properties.agency_id, properties.client_id)` | unique | Lookup `OAuthClient` by client_id at `/token` and `/authorize` |

### `org_oauth_artifacts`

| Index | Type | Purpose |
|---|---|---|
| `(properties.agency_id, properties.token_hash)` | unique | Hot path — every introspection hits this |
| `properties.expires_at` | TTL | Auto-purge expired tokens, codes, revocation records |

### `org_audit_events`

| Index | Type | Purpose |
|---|---|---|
| `(properties.agency_id, properties.event_type)` | non-unique | Filter audit by event class |
| `(properties.agency_id, properties.actor_id)` | non-unique | Audit trail for a user / client |
| `(properties.agency_id, properties.subject_id)` | non-unique | "What happened to entity X" |
| `(properties.agency_id, properties.event_at)` | non-unique (range) | Time-window queries |

### `org_relationships`

The default `_from` / `_to` indexes that ArangoDB creates automatically
for edge collections are sufficient. No additional indexes for v1.

---

## Edge inventory

Forward direction only — entitygraph auto-creates the inverse
`belongs_to_*` edge unless the inverse name is explicitly listed.

| From | Edge | To | Inverse |
|---|---|---|---|
| `Organization` | `has_user` | `User` | `belongs_to_organization` |
| `Organization` | `has_role` | `Role` | `belongs_to_organization` |
| `Organization` | `has_scope` | `Scope` | `belongs_to_organization` |
| `Organization` | `has_oauth_client` | `OAuthClient` | `belongs_to_organization` |
| `Organization` | `has_audit_event` | `AuditEvent` | `belongs_to_organization` |
| `User` | `has_password_credential` | `PasswordCredential` | `belongs_to_user` |
| `User` | `has_membership` | `Membership` | `belongs_to_user` |
| `User` | `has_invitation` | `Invitation` | `belongs_to_user` |
| `Role` | `has_scope` | `Scope` | (multi-source — Role and Organization both source `has_scope`; see note) |
| `Membership` | `grants_role` | `Role` | `granted_by_membership` |
| `Invitation` | `invited_by` | `User` | `has_issued_invitation` |
| `Invitation` | `will_grant_role` | `Role` | `will_be_granted_via_invitation` |
| `OAuthClient` | `has_redirect_uri` | `RedirectURI` | `belongs_to_oauth_client` |
| `OAuthClient` | `has_client_secret` | `ClientSecret` | `belongs_to_oauth_client` |
| `OAuthClient` | `allows_scope` | `Scope` | `allowed_by_oauth_client` |
| `AuthorizationCode` | `issued_to` | `OAuthClient` | `issued_authorization_code` |
| `AuthorizationCode` | `issued_for` | `User` | `received_authorization_code` |
| `AuthorizationCode` | `has_requested_scope` | `Scope` | `requested_by_authorization_code` |
| `AccessToken` | `issued_to` | `OAuthClient` | `issued_access_token` |
| `AccessToken` | `issued_for` | `User` | `received_access_token` |
| `AccessToken` | `has_scope` | `Scope` | `bound_to_access_token` |
| `RefreshToken` | `issued_to` | `OAuthClient` | `issued_refresh_token` |
| `RefreshToken` | `issued_for` | `User` | `received_refresh_token` |
| `RefreshToken` | `has_scope` | `Scope` | `bound_to_refresh_token` |
| `RefreshToken` | `parent` | `RefreshToken` | `child` |

`TokenRevocation` has **no edges** — lookup is by `token_hash`
equality (architecture §4.5; locked in
[data-model/oauth.md](data-model/oauth.md)).

`Role.has_scope` and `Organization.has_scope` use the same edge name.
This is fine — the edge collection is a flat set of `(from, to, type)`
triples; the source entity disambiguates.

---

## Immutability

Set `TypeDefinition.Immutable = true` on every type below; everything
else defaults to mutable.

- `AuthorizationCode`
- `AccessToken`
- `RefreshToken`
- `TokenRevocation`
- `AuditEvent`

All five have `Update*` paths that return `ErrImmutableType`
([error-catalog.md](error-catalog.md)).

---

## Idempotent seed of built-in roles

`DefaultOrgSchema()` declares the four built-in `Role` entity types,
but **does not** seed instances. Instance seeding happens in
`InitOrganization`:

```
For each agency at init:
  For each builtin role name in [super_admin, admin, member, viewer]:
    CreateEntity Role {agency_id, name, builtin: true, ...}  (idempotent)
  For super_admin: CreateRelationship Role.has_scope → every existing
                   org:* and audit:* Scope (also idempotent)
  For admin:      CreateRelationship Role.has_scope → org:admin, audit:read
  For member, viewer: no scope edges (per [role-taxonomy.md](role-taxonomy.md)
                   BR-001 = empty default)
```

**Version-upgrade migration** (BR-002, resolved): when a future
release of CodeValdOrg introduces a new `org:*` or `audit:*` scope,
its migration step must seed the `super_admin ──has_scope──▶ scope`
edge for **every existing agency**. There is no auto-bind on
`RegisterScope`, so the edge does not appear by itself. This is the
single migration responsibility CodeValdOrg accepts in exchange for
the no-auto-bind safety guarantee.

Resource-service scopes (`git:*`, `work:*`, etc.) are **never**
auto-bound to any built-in role at any time. Admins must explicitly
grant them via `GrantScopeToRole` when they want a role to carry
those scopes.

---

## v2 evolution path

The schema evolves by appending to `Types[]`. Already-anticipated
additions:

- `WebAuthnCredential`, `OIDCCredential` (per
  [data-model/identity.md](data-model/identity.md))
- Federation entities (`IdentityProvider`, `FederatedSubject`)
- Optionally: an outbox table (`OAuthOutbox`) if Q26's strong-CP
  contract is downgraded to eventually-consistent

None of these break existing entity types or edge names.
