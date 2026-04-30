# 3 — Software Development

## Overview

This section captures development-oriented artefacts: research gaps,
MVP scope, per-topic implementation details, and task tracking.

The starting point for new work is **[research-gaps.md](research-gaps.md)** —
it maps the documented design (sections 1 & 2) against the build-ready
specifications a Go engineer needs to ship CodeValdOrg, and lists every
outstanding decision or unwritten artefact.

---

## Index

| Document | Description |
|---|---|
| [research-gaps.md](research-gaps.md) | Gap analysis across the 13 research areas defined in `.github/prompts/research.prompt.md`; identifies missing artefacts and the next deliverables that close each gap |
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

## How to use this section

1. Read `research-gaps.md` to find the highest-priority gap relevant to your task
2. For each gap, the document names the next concrete deliverable (a
   `.proto` file, a `schema.go` function, a config table, etc.)
3. New per-topic detail files belong under `mvp-details/{domain}/` once
   the MVP plan exists — see the refactor workflow in
   `.github/prompts/research.prompt.md` for the folder layout rules
   (≤500 lines per file, README ≤300 lines, group by topic not by task ID)
