# 3 — Software Development

## Overview

This section captures development-oriented artefacts: research gaps,
MVP scope, per-topic implementation details, and task tracking.

Research is complete (all 13 areas closed 2026-04-27). The active implementation
backlog lives in **[mvp.md](mvp.md)**. Start there for day-to-day task selection.

---

## Index

| Document | Description |
|---|---|
| [mvp.md](mvp.md) | Active implementation backlog — 21 tasks across P0/P1/P2 priorities with scope, dependencies, and completion workflow |
| [mvp_done.md](mvp_done.md) | Completed tasks log; rows moved here from `mvp.md` on merge |
| [research-gaps.md](research-gaps.md) | Gap analysis across the 13 research areas — all closed 2026-04-27; historical reference |
| [mvp-details/role-taxonomy.md](mvp-details/role-taxonomy.md) | Proposed v1 scope sets for the four built-in roles (`super_admin`, `admin`, `member`, `viewer`); open questions BR-002, BR-004 still to resolve (BR-001, BR-003 closed) |
| [mvp-details/data-model/](mvp-details/data-model/) | Field-level entity specs for all 15 entity types — closes Area 1 of the gap analysis and feeds `schema.go` (Area 8) |
| [mvp-details/scope-model.md](mvp-details/scope-model.md) | Flat scope grammar, registration & deprecation semantics, effective-scope calculation at mint time — closes Area 4 |
| [mvp-details/error-catalog.md](mvp-details/error-catalog.md) | Sentinel → gRPC code → HTTP status → OAuth `error` mapping; audit/metric handling per error; internal-error sanitisation contract — closes Area 9 |
| [mvp-details/configuration.md](mvp-details/configuration.md) | Env vars (required + optional with defaults), one-process-per-agency mode, secret loading policy, startup validation — closes Area 11 |
| [mvp-details/token-issuance.md](mvp-details/token-issuance.md) | Prefixed-token format (`cv_at_…` / `cv_rt_…`), SHA-256 hash-at-rest, per-grant mint sequence, strong-CP persistence-vs-publish contract, refresh-rotation reuse detection — closes Area 2 |
| [mvp-details/introspection.md](mvp-details/introspection.md) | Lookup algorithm, `{active: false}` parity, caller-auth requirement, opt-in caching contract — closes Area 3 |
| [mvp-details/revocation-and-cache.md](mvp-details/revocation-and-cache.md) | Revocation primitives, pub/sub payload + subscriber expectations, bulk-revocation flows, race handling — closes Area 6 |
| [mvp-details/cross-registration.md](mvp-details/cross-registration.md) | `RegisterRequest` worked example, heartbeat lifecycle, OAuth-endpoints-not-proxied rationale, `/.well-known` discovery — closes Area 7 |
| [mvp-details/schema-reference.md](mvp-details/schema-reference.md) | Collection routing, full index manifest, edge inventory, immutability flags, built-in-role seed flow — closes Area 8 (docs); `schema.go` translation is the remaining code task |
| [mvp-details/grpc-api.md](mvp-details/grpc-api.md) | `proto/codevaldorg/v1/org.proto` shape, `buf` toolchain, message conventions, validation annotations — closes Area 10 |
| [mvp-details/testing-strategy.md](mvp-details/testing-strategy.md) | Three layers (unit / integration / conformance), `testcontainers` ArangoDB, full negative-test checklist, NFR-002 benchmark recipe — closes Area 12 |
| [mvp-details/threat-model.md](mvp-details/threat-model.md) | STRIDE per attacker class (A1–A6), mitigation pointers per cell, v2 backlog, accepted residual risks — closes Area 13 |

---

## MVP Status

| Task ID | Title | Status |
|---|---|---|
| ORG-001 | Module Scaffolding | 📋 Not Started |
| ORG-002 | `schema.go` — `DefaultOrgSchema()` | 📋 Not Started |
| ORG-003 | `models.go` — Go value types | 📋 Not Started |
| ORG-004 | `errors.go` — sentinel errors | 📋 Not Started |
| ORG-005 | Proto + codegen | 📋 Not Started |
| ORG-006 | `OrgManager` interface | 📋 Not Started |
| ORG-007 | `internal/config/config.go` | 📋 Not Started |
| ORG-008 | ArangoDB entitygraph backend | 📋 Not Started |
| ORG-009 | Organization + User lifecycle | 📋 Not Started |
| ORG-010 | Token issuance | 📋 Not Started |
| ORG-011 | Token introspection | 📋 Not Started |
| ORG-012 | Token revocation + pub/sub | 📋 Not Started |
| ORG-013 | Scope registration | 📋 Not Started |
| ORG-014 | Role taxonomy + membership | 📋 Not Started |
| ORG-015 | OAuth client management | 📋 Not Started |
| ORG-016 | Cross registration + route registrar | 📋 Not Started |
| ORG-017 | gRPC server handlers | 📋 Not Started |
| ORG-018 | `cmd/server/main.go` startup wiring | 📋 Not Started |
| ORG-019 | Unit tests | 📋 Not Started |
| ORG-020 | Integration tests | 📋 Not Started |
| ORG-021 | Conformance tests | 📋 Not Started |

---

## Execution Order

```
ORG-001 (scaffolding)
  ├── ORG-002 (schema)   ──→ ORG-008 (backend)
  ├── ORG-003 (models)   ──┐
  ├── ORG-004 (errors)   ──┼──→ ORG-006 (interface) ──→ ORG-009..ORG-015 (domain logic)
  ├── ORG-005 (proto)    ──┘                         ──→ ORG-017 (gRPC handlers)
  └── ORG-007 (config)   ──→ ORG-016 (cross reg)    ──→ ORG-018 (main.go)

ORG-019 (unit tests) — parallel, as each domain task lands
ORG-020 (integration) — after ORG-008
ORG-021 (conformance) — after ORG-018
```

---

## Task Detail Files

| File | Tasks |
|---|---|
| [mvp-details/schema-reference.md](mvp-details/schema-reference.md) | ORG-002 — `DefaultOrgSchema()` |
| [mvp-details/data-model/](mvp-details/data-model/) | ORG-003 — models, ORG-002 — TypeDefinitions |
| [mvp-details/error-catalog.md](mvp-details/error-catalog.md) | ORG-004, ORG-017 — error mapping |
| [mvp-details/grpc-api.md](mvp-details/grpc-api.md) | ORG-005, ORG-017 — proto + handlers |
| [mvp-details/configuration.md](mvp-details/configuration.md) | ORG-007 — config |
| [mvp-details/token-issuance.md](mvp-details/token-issuance.md) | ORG-010 — token issuance |
| [mvp-details/introspection.md](mvp-details/introspection.md) | ORG-011 — introspection |
| [mvp-details/revocation-and-cache.md](mvp-details/revocation-and-cache.md) | ORG-012 — revocation |
| [mvp-details/scope-model.md](mvp-details/scope-model.md) | ORG-013 — scope registration |
| [mvp-details/role-taxonomy.md](mvp-details/role-taxonomy.md) | ORG-014 — roles + membership |
| [mvp-details/cross-registration.md](mvp-details/cross-registration.md) | ORG-016 — cross registration |
| [mvp-details/testing-strategy.md](mvp-details/testing-strategy.md) | ORG-019, ORG-020, ORG-021 |
| [mvp-details/threat-model.md](mvp-details/threat-model.md) | Security reference for all tasks |
