# Scope Model

## Purpose

The contract between CodeValdOrg's token issuance and every resource
service's enforcement point. Closes research-gap **Area 4 (Scope
model)** and supplies the runtime semantics that the
[distributed-PDP decision](../../2-SoftwareDesignAndArchitecture/architecture-authorization-model.md)
relies on.

Decided 2026-04-27 across questions Q18–Q21 of the research Q&A.

---

## Grammar

Scope strings are flat — no hierarchy, no wildcards, no implication.
`git:write` does **not** imply `git:read`; tokens carry every scope
they need explicitly.

| Property | Rule |
|---|---|
| Format | `<service>:<action>` — exactly one colon |
| Characters | `[a-z0-9_]` — lowercase ASCII letters, digits, underscore only |
| Total length | 1–50 characters |
| Service prefix | 1–20 chars; must equal the `registered_by` field on the `Scope` entity |
| Action suffix | 1–28 chars (50 minus prefix max minus colon) |
| Reserved prefixes | `org` and `audit` — only CodeValdOrg may register scopes with these prefixes |
| Forbidden | uppercase, dots, dashes, slashes, spaces, multi-colon, empty halves |

Examples:

| String | Valid? | Reason |
|---|---|---|
| `git:read`, `work:write`, `org:admin`, `audit:read` | yes | — |
| `Git:Read` | no | uppercase forbidden |
| `git:` / `:read` | no | empty half |
| `git::read` | no | exactly one colon |
| `git:repo:read` | no | exactly one colon |
| `git-read` | no | colon required |
| `agency:read` registered by `codevaldorg` | no | service prefix must equal `registered_by` (must be `codevaldagency`) |

The two-part-only constraint forecloses a future "let's add finer
granularity by adding segments" decision that would re-open hierarchy
semantics. If finer granularity is ever needed, it goes in via
underscore-compounded actions (`git:repo_read`, `git:branch_read`),
**not** by adding segments.

---

## Registration — `OrgService.RegisterScope`

Idempotent-on-startup. Resource services blindly call
`RegisterScope` for every scope they own on every boot.

| Pre-existing state | Behaviour |
|---|---|
| Scope does not exist | Create it (`registered_by = caller`, `created_at = now`) — return success |
| Scope exists, `registered_by == caller` | Idempotent update — `description` and `updated_at` refreshed; `created_at` and `deprecated_at` preserved — return success |
| Scope exists, `registered_by != caller` | Refuse with `ErrScopeNameCollision` — service should refuse to start |
| Scope exists, `deprecated_at != null`, `registered_by == caller` | Allowed — un-deprecates by clearing `deprecated_at`. Lets a service walk back a deprecation. |

**Caller identity.** `registered_by` is taken from the bearer token's
`client_id` claim — i.e. the calling resource service uses its own
confidential OAuth client (issued at platform setup) to authenticate
the `RegisterScope` call. The service name is **not** trusted from any
request body field.

This means: every resource service has a confidential `OAuthClient`
in CodeValdOrg, with its `client_id` matching the `registered_by`
prefix it claims (e.g. `codevaldgit` for Git's scopes). Provisioning
this client is part of platform setup, not service runtime.

**No auto-binding to roles** (BR-002, resolved). `RegisterScope`
creates only the `Scope` entity. It does not create any
`Role ──has_scope──▶ Scope` edges — not even for `super_admin`.
A freshly registered scope grants nothing until an admin explicitly
binds it via `GrantScopeToRole(role_id, scope_id)`, or until a
seeding step at agency-init / version-upgrade time creates the edge.

This is the safest posture: nothing is granted until the agency says
so. The trade-off is first-run friction — every new resource service
deployment requires admin work before its scopes carry authority. It
is accepted as the price of explicit-by-default authority.

---

## Deprecation — `OrgService.DeprecateScope`

Sets `deprecated_at = now` on the scope. Existing role bindings keep
working; the admin UI surfaces the deprecation as a warning.

There is **no `DeleteScope` in v1.** Once a scope ID has ever been
issued in a token, deleting it would un-explain past audit records.
Deprecation is permanent-by-default; un-deprecation is the explicit
walk-back primitive (above).

---

## Effective scope calculation at mint time

The Token endpoint computes the effective scope set as a strict
intersection. Bound to the `AccessToken` (and `RefreshToken`) via
`has_scope` edges at mint time, frozen for the token's lifetime.

### Authorization Code + PKCE (interactive — has a user)

```
effective = requested
          ∩ scopesGrantedToUser  (union over User → Memberships → Roles → Scopes)
          ∩ client.allowedScopes (the OAuthClient's allows_scope edges)
          ∩ { s : s.deprecated_at == null }
```

### Client Credentials (no user)

```
effective = requested
          ∩ client.allowedScopes
          ∩ { s : s.deprecated_at == null }
```

### Edge cases

| Case | Behaviour |
|---|---|
| `requested` is empty | Defaults to the maximum: `client.allowedScopes ∩ scopesGrantedToUser ∩ {not deprecated}`. (Or `client.allowedScopes ∩ {not deprecated}` for client-credentials.) RFC 6749 §3.3 calls this "service-defined". |
| `effective` is empty after intersection | Token request **fails** with OAuth error `invalid_scope` (RFC 6749 §5.2). No zero-scope tokens are ever minted. |
| User is `suspended` | All Memberships are treated as inactive at mint time → `scopesGrantedToUser` is empty → `effective` is empty → `invalid_scope`. (Existing tokens are revoked separately by the suspend flow — not narrowed.) |
| Scope was deprecated *after* a token was minted | The token continues to carry and be honoured for the deprecated scope until its natural expiry. New tokens won't include it. |
| Membership / role change after mint | Has **no effect** on live tokens. Revocation is the only narrowing primitive — see [research-gap Area 6](../research-gaps.md). |

---

## Default behaviour for unknown scopes

**Deny-by-default.** The distributed PDP at every resource service
treats an unknown scope name as "not granted" — the request fails the
authorization check. There is no concept of an "unscoped allow".

If a resource service receives a token containing a scope string it
doesn't recognise (e.g. minted before the resource service deployed a
schema-aware build), it ignores that scope — it does not error or
bypass enforcement. Only the scopes the resource service actively
requires for the requested operation matter.

---

## Implications for role-taxonomy

Closed by Q18 (flat grammar, no wildcards):

- **`super_admin` cannot use `org:*`.** It owns every individual
  `org:` scope as separate `Role ──has_scope──▶ Scope` edges. When
  CodeValdOrg registers a new `org:` scope, it must also seed an edge
  from the `super_admin` role of every existing agency. This is a
  schema-evolution responsibility, not a runtime wildcard.
- **BR-003 (wildcard expansion timing) is moot.** No wildcards exist.
- **BR-002 (auto-bind on register) — resolved**: pure manual. See
  the "No auto-binding to roles" section above and
  [role-taxonomy.md](role-taxonomy.md). Every `Role ──has_scope──▶ Scope`
  edge is created either (i) by an admin via `GrantScopeToRole`, or
  (ii) by a seeding step at agency-init / version-upgrade time. Never
  as a side-effect of `RegisterScope`.

---

## Source-of-truth handoff

- `OrgService.RegisterScope` and `OrgService.DeprecateScope` are
  finalised in [research-gap Area 10 (gRPC proto definition)](../research-gaps.md).
- The token-mint flow (`OrgService.Token`) wires the effective-scope
  calculation into [research-gap Area 2 (Token issuance flow)](../research-gaps.md).
- Distributed PDP enforcement at every resource service is governed by
  [architecture-authorization-model.md](../../2-SoftwareDesignAndArchitecture/architecture-authorization-model.md).
