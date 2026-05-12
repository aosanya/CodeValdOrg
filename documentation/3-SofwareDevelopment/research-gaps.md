# CodeValdOrg — Research & Documentation Gap Analysis

## Purpose

This document maps the 13 research areas defined in
[`.github/prompts/research.prompt.md`](../../.github/prompts/research.prompt.md)
against the documentation that exists today
([1-SoftwareRequirements](../1-SoftwareRequirements/) and
[2-SoftwareDesignAndArchitecture](../2-SoftwareDesignAndArchitecture/)),
and identifies every artefact a Go engineer would still need to start
building the service.

Each section follows the same shape:

- **What is documented** — a one-line summary of current coverage
- **Gaps** — specific unanswered questions or missing artefacts
- **Next deliverable** — the concrete file/spec that closes the gap

A summary table is given first, ordered by priority.

---

## Summary Table

| # | Research Area | Coverage | Priority | Next Deliverable |
|---|---|---|---|---|
| 1 | Principal & key data model | **Closed (2026-04-27)** — see [mvp-details/data-model/](mvp-details/data-model/) | — | (closed) |
| 8 | ArangoDB schema | **Closed at doc level (2026-04-27)** — see [mvp-details/schema-reference.md](mvp-details/schema-reference.md). Code task remaining: write `schema.go`. | code-only | `DefaultOrgSchema()` in `schema.go` |
| 10 | gRPC proto definition | **Closed (2026-04-27)** — see [mvp-details/grpc-api.md](mvp-details/grpc-api.md) | — | (closed) |
| 4 | Scope model | **Closed (2026-04-27)** — see [mvp-details/scope-model.md](mvp-details/scope-model.md) | — | (closed) |
| 5 | Authorization decision flow | **Closed (2026-04-27)** — see [architecture-authorization-model.md](../2-SoftwareDesignAndArchitecture/architecture-authorization-model.md) | — | (closed) |
| 9 | Error handling | **Closed (2026-04-27)** — see [mvp-details/error-catalog.md](mvp-details/error-catalog.md) | — | (closed) |
| 11 | Configuration | **Closed (2026-04-27)** — see [mvp-details/configuration.md](mvp-details/configuration.md) | — | (closed) |
| 12 | Testing strategy | **Closed (2026-04-27)** — see [mvp-details/testing-strategy.md](mvp-details/testing-strategy.md) | — | (closed) |
| 2 | Token issuance flow | **Closed (2026-04-27)** — see [mvp-details/token-issuance.md](mvp-details/token-issuance.md) | — | (closed) |
| 3 | Introspection (verification) flow | **Closed (2026-04-27)** — see [mvp-details/introspection.md](mvp-details/introspection.md) | — | (closed) |
| 6 | Token rotation & revocation | **Closed (2026-04-27)** — see [mvp-details/revocation-and-cache.md](mvp-details/revocation-and-cache.md) | — | (closed) |
| 7 | Cross registration | **Closed (2026-04-27)** — see [mvp-details/cross-registration.md](mvp-details/cross-registration.md) | — | (closed) |
| 13 | Threat model | **Closed (2026-04-27)** — see [mvp-details/threat-model.md](mvp-details/threat-model.md) | — | (closed) |

---

## 1. Principal & Key Data Model — **CLOSED (2026-04-27)**

Resolved through the iterative Q&A on 2026-04-27. The full field-level
spec lives in
[mvp-details/data-model/](mvp-details/data-model/), split by topic:

- [identity.md](mvp-details/data-model/identity.md) — `Organization`, `User`, `PasswordCredential`, `Role`, `Scope`, `Membership`, `Invitation`
- [oauth.md](mvp-details/data-model/oauth.md) — `OAuthClient`, `ClientSecret`, `RedirectURI`, `AuthorizationCode`, `AccessToken`, `RefreshToken`, `TokenRevocation`
- [audit.md](mvp-details/data-model/audit.md) — `AuditEvent`

Key decisions captured along the way:

- Every field is a typed `PropertyDefinition`; no freeform `attributes`
  map (memory: `feedback_codevaldorg_schema_properties`).
- Two tiers per entity — Core (`Required: true` + indexed) vs Long
  tail (`Required: false`, unindexed).
- Credentials are a separate entity type per kind; v1 ships
  `PasswordCredential` only. `WebAuthnCredential` / `OIDCCredential`
  arrive in v2 by appending to `Types[]`.
- Client-secret rotation grace is modelled as multiple `ClientSecret`
  entities under one `OAuthClient`, mirroring `PasswordCredential`.
- Redirect URIs are entities, not a JSON-serialised array property.
- OAuth-artifact tokens hash with SHA-256 (high-entropy random bytes);
  user-chosen secrets hash with Argon2id PHC.
- `TokenRevocation` is a hash-keyed parallel-lookup record with **no
  edge** to the revoked token entity.
- `AuditEvent` carries structured details in a single typed `payload`
  string property, per the entitygraph telemetry/event convention.

---

## 2. Token Issuance Flow — **CLOSED (2026-04-27)**

Resolved by [mvp-details/token-issuance.md](mvp-details/token-issuance.md).

**Decisions:**

- **Prefixed tokens** (`cv_ac_…` / `cv_at_…` / `cv_rt_…` / `cv_iv_…`)
  to enable leak-detector scanning (Q27, Stripe/GitHub-style
  convention).
- **256-bit random tail** from `crypto/rand`, base64url-encoded.
- **SHA-256 hash at rest** of the *full prefixed plaintext* (not just
  the random tail) — defence-in-depth against cross-collection
  lookup bugs.
- **No `crypto/subtle` compare** needed — lookup is hash equality, not
  pairwise compare.
- **Strong-CP publish-or-roll-back** (Q26 option b). DB write + Cross
  publish must both succeed; any failure → 503
  `temporarily_unavailable`. Compensating-delete failures are
  harmless because the unguessable orphan token TTL-purges.
- **Effective scope** computed by intersection per
  [scope-model.md](scope-model.md), bound to the token via
  `has_scope` edges, frozen for the token lifetime.
- **Refresh-rotation reuse detection** walks the `parent` chain and
  revokes every ancestor + descendant on detected reuse (FR-005,
  architecture §5.3).
- **Cross event payload** carries IDs and metadata only — never
  plaintext.

---

## 3. Token Verification (Introspection) Flow — **CLOSED (2026-04-27)**

Resolved by [mvp-details/introspection.md](mvp-details/introspection.md).
Lookup is hash equality through the unique index — no
`crypto/subtle` compare needed. `{active: false}` parity for unknown /
expired / revoked / malformed (RFC 7662). Caller authentication is
required (rejects A1 anonymous probing). Caching contract is opt-in
for v1 with mandatory invalidation on `org.token.revoked`.

---

## 4. Scope Model — **CLOSED (2026-04-27)**

Resolved by [mvp-details/scope-model.md](mvp-details/scope-model.md).

**Decisions:**

- **Flat grammar.** `<service>:<action>` exactly. `[a-z0-9_]` only,
  1–50 chars, exactly one colon. No wildcards, no hierarchy, no
  implication. `git:write` does not imply `git:read`.
- **Reserved prefixes** `org` and `audit` belong to CodeValdOrg.
- **`registered_by` is taken from the bearer token's `client_id`** —
  every resource service has its own confidential `OAuthClient` and
  authenticates `RegisterScope` calls through it; the service name is
  not trusted from any request body.
- **Idempotent-on-startup registration**: same-owner re-register is a
  no-op update; different-owner is `ErrScopeNameCollision`.
- **No `DeleteScope` in v1**; deprecation is permanent-by-default,
  un-deprecation is the explicit walk-back primitive.
- **Effective scope = strict intersection** of `requested ∩
  scopesGrantedToUser ∩ client.allowedScopes ∩ {not deprecated}`,
  bound to the token via `has_scope` edges and frozen at mint time.
  Empty intersection → `invalid_scope` (RFC 6749 §5.2). Live tokens
  do not narrow on subsequent role/membership/client changes —
  revocation is the only narrowing primitive.
- **Deny-by-default** for unknown scopes at every resource server.

The choice of flat grammar makes the role-taxonomy "wildcard"
question (BR-003) moot and escalates BR-002 (auto-bind on register)
to load-bearing — see [mvp-details/role-taxonomy.md](mvp-details/role-taxonomy.md).

---

## 5. Authorization Decision Flow — **CLOSED (2026-04-27)**

Resolved by
[architecture-authorization-model.md](../2-SoftwareDesignAndArchitecture/architecture-authorization-model.md).

**Decision:** v1 uses a **distributed PDP** — CodeValdOrg owns
identity and token issuance only; every resource service is its own
PDP, comparing the requested operation against the `scope[]` claim
returned by `Introspect`. No `OrgService.Authorize` RPC exists in v1.

The decision document captures the trade-offs accepted, what
CodeValdOrg explicitly does *not* own, and three re-visit triggers
that would push the model toward a hybrid (Introspect + opt-in
`Authorize`) design.

Outstanding sub-questions deferred:

- "Default behaviour for unknown scopes" — deny-by-default is the
  industry standard and follows from distributed-PDP enforcement; it
  will be locked in the scope-grammar spec (Area 4).
- "Scope-to-action examples per resource service" — owned by each
  resource service's docs, not CodeValdOrg's. Out of scope here.

---

## 6. Token Rotation & Revocation — **CLOSED (2026-04-27)**

Resolved by
[mvp-details/revocation-and-cache.md](mvp-details/revocation-and-cache.md).
Revocation is at-least-once via `org.token.revoked`;
subscribers are idempotent on `token_hash`. `TokenRevocation.expires_at`
matches the underlying token's natural expiry so the TTL purges
revocation records exactly when they stop being load-bearing. Race on
refresh-rotation reuse is resolved by the unique index on
`(agency_id, token_hash)` — no locking. `SuspendUser` and
`DisableOrganization` are bulk-revocation primitives.

---

## 7. Cross Registration — **CLOSED (2026-04-27)**

Resolved by [mvp-details/cross-registration.md](mvp-details/cross-registration.md).
Worked `RegisterRequest` example with all 27 admin routes; OAuth
endpoints are deliberately NOT in `Routes[]` (issuer-URL contract).
Heartbeat re-sends the full payload every `ORG_REGISTRAR_INTERVAL`;
Cross removes the route after missing N heartbeats (Cross-side
config). Discovery via `<ORG_ISSUER_URL>/{agencyId}/.well-known/oauth-authorization-server`,
served directly by Org. Version negotiation deferred — v2-breaking
change uses a new `ServiceName`.

---

## 8. ArangoDB Schema — **CLOSED at doc level (2026-04-27)**

Resolved at the documentation level by
[mvp-details/schema-reference.md](mvp-details/schema-reference.md):
collection routing, full index manifest (TTL on
`org_oauth_artifacts.expires_at`, unique compounds on
`(agency_id, email)`, `(agency_id, name)`, `(agency_id, client_id)`,
`(agency_id, token_hash)`), edge inventory with auto-inverse names,
immutability flags, and the built-in-role seeding flow.

**Code task remaining:** translate this reference into `schema.go` at
the repo root, implementing `DefaultOrgSchema()` — matches the
pattern of [CodeValdGit/schema.go](../../../CodeValdGit/schema.go).
Mechanical work; no further research required.

---

## 9. Error Handling — **CLOSED (2026-04-27)**

Resolved by [mvp-details/error-catalog.md](mvp-details/error-catalog.md).

**Decisions:**

- Three handling labels per row — `audit-per-occurrence`,
  `metric-count`, or `audit + metric`. Auth-relevant denials are
  audited per-occurrence; the noisy `invalid_request` class is metric-only
  (decided in Q22, to keep the audit log forensically useful and out
  of the DB-pollution failure mode).
- Internal-error sanitisation: every error not in the catalog becomes
  `codes.Internal` / `500` `server_error` with a correlation ID; the
  real error string + stack trace stays in logs and never crosses the
  trust boundary (decided in Q23).
- All sentinels live in `errors.go`; `internal/server/mappers.go` is
  the only place that translates sentinel → gRPC code / HTTP status /
  OAuth error body.
- Two follow-ups noted (audit `payload` shape, metric label cardinality
  rules) — not blocking.

---

## 10. gRPC Proto Definition — **CLOSED (2026-04-27)**

Resolved by [mvp-details/grpc-api.md](mvp-details/grpc-api.md). Full
RPC list, message-shape conventions (plurals over filter sub-messages,
`agency_id` always field 1, cursor pagination via
`page_size` + `page_token`, `google.protobuf.Timestamp` over RFC 3339
strings on the wire), `buf` toolchain setup, `protoc-gen-validate`
annotations for emails / scopes / redirect URIs. All RPCs unary; no
streaming in v1. `ListAuditEvents` is a thin wrapper over
`entitygraph.ListEntities` per the telemetry/event memory rule.

---

## 11. Configuration — **CLOSED (2026-04-27)**

Resolved by [mvp-details/configuration.md](mvp-details/configuration.md).

**Decisions:**

- **One process per agency** (Q24). `AGENCY_ID` is baked in at
  startup; rejects RPCs whose `agencyId` path parameter doesn't
  match. Matches CodeValdGit's per-repo isolation model.
- 6 required env vars + 12 optional with defaults; full table in the
  mvp-details doc.
- Plain env vars only — no Vault, no `_FILE` variants, no YAML
  overlay (twelve-factor by choice; v2 problem otherwise).
- Two-pass startup validation: required-set check then
  range/format check; misconfigured process never reaches a
  half-started state.
- Five items deliberately deferred: TLS termination (at ingress, not
  Org), separate `HEALTH_ADDR`, rate-limit knobs, YAML overlay,
  per-agency runtime config.

---

## 12. Testing Strategy — **CLOSED (2026-04-27)**

Resolved by [mvp-details/testing-strategy.md](mvp-details/testing-strategy.md).
Three layers: unit (in-memory `DataManager` double, `CrossPublisher`
mock), integration (`testcontainers-go` ArangoDB, per-test database),
conformance (RFC 6749 / 7636 / 7662 / 7009 / 8414, OIDF fixtures
where licensing permits). Coverage targets (≥85–95% per package),
NFR-002 benchmark recipe, full negative-test checklist (per OAuth
flow + admin + schema invariants), CI matrix gating merges on unit +
integration; conformance gates only `main`.

---

## 13. Threat Model — **CLOSED (2026-04-27)**

Resolved by [mvp-details/threat-model.md](mvp-details/threat-model.md).
STRIDE matrix per attacker class A1–A6: network adversary, compromised
end-user device, malicious OAuth client developer, compromised
resource server, operator with read-only DB, operator with read/write
DB (last is trusted-by-design). Mitigations point to specific
sections; OPEN cells flag rate limiting (deferred to v2) and
browser-resident tokens (accepted residual). v2 backlog and explicit
accepted risks listed at the end.

---

## Cross-cutting observations (historical — preserved for context)

Two patterns drove the order in which the Q&A walked through the gaps:

1. **The biggest single-file deliverable is `schema.go`** — it locks
   entity fields (gap 1), collection layout and indexes (gap 8), and
   is the upstream source of truth for the proto messages (gap 10) and
   the model types referenced by every other gap. As of 2026-04-27
   this is the only remaining task — see
   [mvp-details/schema-reference.md](mvp-details/schema-reference.md)
   for the field-level reference and translate into `schema.go`.

2. **The `OrgService` interface is silent on a central PDP** — meaning
   resource services authorise themselves using only the `scope` claim.
   This load-bearing decision (gap 5) was pinned first, before the
   scope-model spec (gap 4) and introspection caching contract (gap 6),
   to avoid downstream rework. Recorded in
   [architecture-authorization-model.md](../2-SoftwareDesignAndArchitecture/architecture-authorization-model.md)
   with explicit re-visit triggers.

## Status

All 13 areas closed at the documentation level on 2026-04-27 through
the iterative single-question Q&A defined in
[`research.prompt.md`](../../.github/prompts/research.prompt.md).

Remaining open work tracked outside this gap analysis:

- **Code task — `schema.go`** — translate
  [mvp-details/schema-reference.md](mvp-details/schema-reference.md)
  into `DefaultOrgSchema()` at the repo root.
- **Code task — `proto/codevaldorg/v1/org.proto`** — write the proto
  file per [mvp-details/grpc-api.md](mvp-details/grpc-api.md) and
  generate stubs into `gen/go/codevaldorg/v1/`.
- **v2 commitments** — DPoP token binding (RFC 9449) is a documented
  v2 deliverable with the surface scope laid out in
  [mvp-details/threat-model.md](mvp-details/threat-model.md).

Closed in the 2026-04-28 follow-up session:

- ✓ BR-002 (auto-bind on `RegisterScope`) — pure manual, no auto-binding
- ✓ BR-004 (custom-role inheritance) — flat for v1; deferred to v2
- ✓ Rate limiting on OAuth endpoints — in-process per-`client_id` token-bucket limiter
- ✓ Browser-resident token mitigation — accepted residual for v1; DPoP committed for v2
- ✓ Clock injection for tests — `Clock` interface plumbed through
- ✓ `crypto/rand` failure handling — `ErrTemporarilyUnavailable`, no fallback PRNG
- ✓ Per-OAuthClient bulk revocation on client deletion — synchronous chunked revoke before soft-delete

---

## How to read this document

- Pick the highest-priority gap relevant to your current task.
- The "Next deliverable" line names the concrete file to write next;
  follow the refactor workflow in
  [research.prompt.md](../../.github/prompts/research.prompt.md) so
  individual files stay ≤500 lines and topic-grouped.
- When closing a gap, update the "Coverage" cell in the summary table
  here; remove the section once the gap is fully addressed and the
  resulting document is linked from `2-SoftwareDesignAndArchitecture/`
  or `3-SofwareDevelopment/mvp-details/`.
