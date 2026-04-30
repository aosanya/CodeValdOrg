# 2 — Software Design & Architecture

## Overview

This section captures the **how** — design decisions, data model, component architecture, and OAuth 2.0 flow implementation for CodeValdOrg.

---

## Index

| Document | Description |
|---|---|
| [architecture.md](architecture.md) | Core design decisions, per-agency storage, OAuth 2.0 flow design, entity schema, gRPC / HTTP surface, integration via CodeValdCross |
| [architecture-authorization-model.md](architecture-authorization-model.md) | v1 distributed-PDP decision — CodeValdOrg owns issuance only; resource services compare requested operations against the `scope[]` claim. Re-visit triggers documented |

---

## Key Design Decisions at a Glance

| Decision | Choice | Rationale |
|---|---|---|
| Identity protocol | OAuth 2.0 + OAuth 2.1 tightenings | Industry standard; every SDK has a client; documented threat model |
| Storage granularity | One Organization per Agency, per-agency database | Matches CodeValdGit and CodeValdAgency; hard tenant isolation |
| Storage backend | ArangoDB via `entitygraph.DataManager` | Consistent with the rest of the platform; schema is declared, not migrated |
| Access token format (v1) | Opaque, server-validated via introspection | Simple revocation; no signing-key rotation needed at launch |
| Access token format (v2 — optional) | Signed JWT following RFC 9068 | Zero-round-trip validation once revocation story is settled |
| PKCE policy | Mandatory for all clients; `S256` only | OAuth 2.1 alignment; `plain` method is rejected |
| Redirect URI matching | Exact string match | Prevents open-redirect attacks |
| Refresh token policy | Rotating, single-use | Defangs stolen-token replay; revokes ancestor chain on reuse detection |
| Client secret hashing | Argon2id at rest | Plaintext returned once on create/rotate |
| Admin surface | gRPC `OrgService`, exposed via CodeValdCross HTTP proxy | Same pattern as CodeValdGit / CodeValdAgency |
| Transport | gRPC + HTTP multiplexed on one port via `cmux` | Same pattern as CodeValdGit Smart HTTP |
| Service registration | `internal/registrar` heartbeat to Cross every 20 s | Zero-recompile route publishing |
