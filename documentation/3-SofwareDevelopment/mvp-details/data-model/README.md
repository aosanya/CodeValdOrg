# Data Model

## Purpose

Field-level entity specifications for CodeValdOrg, derived from the
research-gap Q&A run on 2026-04-27. Together these documents close
**Area 1 (Principal & key data model)** of
[research-gaps.md](../../research-gaps.md) and supply most of the input
for **Area 8 (ArangoDB schema)** — what's left for Area 8 is the actual
`schema.go` translation plus the index manifest.

---

## Entities at a glance

15 entity types across three topical groupings:

| Topic | Entities | Storage collection | File |
|---|---|---|---|
| Identity | `Organization`, `User`, `PasswordCredential`, `Role`, `Scope`, `Membership`, `Invitation` | `org_entities` | [identity.md](identity.md) |
| OAuth | `OAuthClient`, `ClientSecret`, `RedirectURI`, `AuthorizationCode`, `AccessToken`, `RefreshToken`, `TokenRevocation` | `org_oauth_clients` (mutable), `org_oauth_artifacts` (immutable, TTL-indexed) | [oauth.md](oauth.md) |
| Audit | `AuditEvent` | `org_audit_events` | [audit.md](audit.md) |

All edges are stored in `org_relationships`.

---

## Schema rules in force

These rules govern every entity in the model and are why each topic
file looks the way it does:

- **Every field is a typed `types.PropertyDefinition`.** No
  hand-rolled Go structs, no freeform `attributes` map. (See
  `feedback_codevaldorg_schema_properties` memory.)
- **Two tiers per entity:**
  - **Core** — `Required: true`, backed by an index. Always includes
    `agency_id`, plus per-entity uniqueness keys.
  - **Long tail** — declared as `PropertyDefinition` but
    `Required: false`, unindexed. Timestamps, optional profile fields.
- **Lists are not properties.** A list of references becomes edges to
  separate entities (e.g. `OAuthClient ──has_redirect_uri──▶ RedirectURI`,
  `Invitation ──will_grant_role──▶ Role`). The two narrow exceptions:
  - Fixed enum lists use `PropertyTypeMultiSelect` (e.g.
    `OAuthClient.allowed_grant_types`).
  - Event payloads use a single `payload` `PropertyTypeString` holding
    JSON, by the entitygraph telemetry/event convention (see
    `feedback_no_separate_telemetry_event_types` memory).
- **Timestamps are `PropertyTypeString` (RFC 3339).** Matches the
  CodeValdGit precedent — `PropertyTypeDatetime` is available in
  SharedLib but not used by sibling services.
- **Hashes:**
  - User-chosen secrets (passwords, client secrets) → **Argon2id PHC**
  - High-entropy random tokens (auth codes, access/refresh tokens,
    invitation tokens) → **SHA-256, base64url**. No `crypto/subtle`
    compare needed because lookup is by hash equality, not pairwise.
- **Immutability** is declared via `TypeDefinition.Immutable: true`
  and applies to all OAuth artifacts and `AuditEvent`. `Update*` paths
  return `ErrImmutableType`.

---

## Conventions for cross-entity layout

- **Collection routing** is via `TypeDefinition.StorageCollection`.
  Mutable identity entities share `org_entities`; OAuth artifacts share
  `org_oauth_artifacts` (TTL-indexed for auto-purge); audit events
  live in `org_audit_events`. No bespoke per-entity collections.
- **Inverse edges** are auto-created by
  `entitygraph.DataManager.CreateRelationship` — declare the forward
  direction only; the documented `belongs_to_*` inverses are implicit.
- **Uniqueness within a parent** (e.g. redirect URIs unique per
  OAuthClient) is **service-layer enforced**, not DB-indexed —
  matching the CodeValdGit precedent which never denormalises parent
  IDs onto children.
- **Soft-delete** uses a `deleted_at` timestamp; `disabled_at` is a
  *separate* lifecycle state for compliance holds (reversible). Never
  the same field.

---

## Open items

- **Built-in role scope sets** — see
  [role-taxonomy.md](../role-taxonomy.md) — BR-002 / BR-003 / BR-004
  still open.
- **`ListAuditEvents` RPC shape** — should be a thin wrapper over
  `ListEntities(typeID="audit_event")`, not a bespoke storage path.
  Tracked in research-gaps.md Area 10 (gRPC proto definition).
