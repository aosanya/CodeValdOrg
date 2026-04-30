# Built-in Role Taxonomy (v1 starting point)

## Purpose

[FR-003](../../1-SoftwareRequirements/requirements.md) names four
built-in roles (`super_admin`, `admin`, `member`, `viewer`) but does
not specify what each one is allowed to do. This document is the
**proposed starting taxonomy** — concrete enough to push back on, loose
enough to evolve once resource services begin registering their scopes
(FR-008).

This doc is the source of truth for the **scope set bound to each
built-in role at agency provisioning time**. It will be referenced by
`schema.go` (when seeding default Role entities) and by the scope-model
spec (gap area 4 — still open).

> **Status:** PROPOSED — not yet ratified. Open questions are listed
> at the bottom; resolve them before implementing.

---

## Taxonomy

| Role | What it can do | Typical scopes |
|---|---|---|
| `super_admin` | Everything, including destructive org-level ops (delete the Organization) | Every individual `org:*` scope, bound as separate `has_scope` edges. Flat grammar (see [scope-model.md](scope-model.md)) means there is no wildcard — when CodeValdOrg registers a new `org:` scope it must also seed an edge from every existing agency's `super_admin` role |
| `admin` | Manage users, roles, OAuth clients, read audit log — but **cannot** delete the Organization | `org:admin`, `audit:read` |
| `member` | Regular contributor — uses the platform's resource services to do work, but no admin authority | Resource scopes registered by other services (e.g. `git:write`, `work:write`, `agency:read`); **no `org:*` scopes** |
| `viewer` | Read-only across resources — for stakeholders / auditors who shouldn't change anything | Resource read scopes only (`git:read`, `work:read`, `agency:read`) |

---

## Notes per role

### `super_admin`

- The only role that can delete the Organization.
- Holds every individual `org:*` scope as a separate
  `has_scope` edge — there is **no** wildcard scope (flat grammar; see
  [scope-model.md](scope-model.md)).
- **Bindings are explicit, not automatic** (BR-002, resolved). When
  CodeValdOrg adds a new `org:` scope in a future release, it must
  seed the `super_admin ──has_scope──▶ scope` edge for every existing
  agency as part of the version-upgrade migration. New agency
  provisioning seeds the edges as part of `InitOrganization`. There
  is no auto-bind on `RegisterScope`.
- There must be at least one `super_admin` per agency at all times;
  the last one cannot be demoted (enforced at the service layer).

### `admin`

- Day-to-day ops role. Can:
  - Invite, suspend, soft-delete users
  - Create / update / delete custom roles
  - Create / rotate / revoke OAuth clients
  - Read the audit log
- Cannot:
  - Delete the Organization
  - Read or write any resource service data directly (no `git:*`,
    `work:*`, etc. — the admin role exists for *org* administration,
    not platform usage)

### `member`

- The **default working role** — a regular logged-in user who isn't an
  admin and is allowed to use Git / Work / Agency at the level the
  agency's policy permits.
- Owns no `org:*` scopes.
- Its exact scope set is determined by which resource services have
  registered scopes (see Caveat below) and which of those scopes the
  agency has chosen to bind to `member`.

### `viewer`

- For stakeholders, auditors, observers who shouldn't change anything.
- Read-only across all resource services that have registered a `:read`
  scope.
- Owns no `org:*` scopes and no write/admin scopes.

---

## Caveat — resource scopes are dynamic

Apart from `org:*` and `audit:read`, every scope referenced above is
**registered by a resource service** at startup via
`OrgService.RegisterScope`. CodeValdOrg itself does not own those scope
strings — it just persists them.

This means a built-in role's effective set is a function of:

1. The fixed `org:*` / `audit:*` scopes CodeValdOrg owns
2. Plus whichever resource scopes the agency has chosen to bind to the
   role at any given time

If a resource service hasn't registered yet, a role can't grant its
scopes. This is by design — it forces explicit registration before any
authorisation decision can be made.

---

## Open questions

| # | Question | Status | Why it matters |
|---|---|---|---|
| BR-001 | At agency provisioning time, do built-in roles ship pre-bound to a default scope set, or empty? | **Resolved (2026-04-27): empty.** Built-in roles are seeded with their `org:*` / `audit:*` scopes only (per the Taxonomy table); no resource scopes are auto-bound. Admins bind resource scopes explicitly to `member` / `viewer` after each resource service registers. | Safer security posture — nothing is granted until the agency says so; costs first-run friction |
| BR-002 | When a new resource service registers a scope, does it auto-bind to any built-in role, or is binding always explicit? | **Resolved (2026-04-27): pure manual.** `RegisterScope` creates only the `Scope` entity; no role bindings are created. Even `super_admin` does not auto-get new scopes. Bindings happen via explicit admin action (`GrantScopeToRole`) or at agency-init / version-upgrade seed time. | Safest posture — no surprises; first-run friction accepted as the price of explicit-by-default authority |
| BR-003 | Is the wildcard `org:*` resolved at issuance time or at evaluation time? | **Resolved (2026-04-27): MOOT.** Flat grammar (see [scope-model.md](scope-model.md)) has no wildcards — the question doesn't arise | — |
| BR-004 | Can custom roles inherit from a built-in (e.g. "like `member` but also gets `billing:read`") via a `parent_role` edge? | **Resolved (2026-04-27): no, flat for v1.** Every custom role enumerates its full scope set as direct `has_scope` edges. Inheritance is purely admin ergonomics — distributed-PDP enforcement sees only the flat scope set on the introspected token, so adding `parent_role` later is non-breaking. Deferred to v2. | Cost of starting flat is admin verbosity; cost of inheritance edge cases (cycles, parent deletion, conflicting deprecation) is higher than v1 needs |

---

## Source-of-truth handoff

Once questions BR-001 → BR-004 are resolved, the agreed scope sets
move into:

- `schema.go` — `DefaultOrgSchema()` seeds the four built-in `Role`
  entities and their `has_scope` edges at agency provisioning time
- `mvp-details/scope-model.md` (not yet written — gap area 4) — the
  scope grammar, wildcard rules, and registration semantics that
  underpin this taxonomy
