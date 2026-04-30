# 1 — Software Requirements

## Overview

This section captures everything **what** CodeValdOrg must do and **why** — without prescribing how.

---

## Index

| Document | Description |
|---|---|
| [requirements.md](requirements.md) | Functional requirements (FR-001–FR-010), non-functional requirements, scope, and open questions |
| [introduction/problem-definition.md](introduction/problem-definition.md) | Problem statement and motivation for the service |
| [introduction/high-level-features.md](introduction/high-level-features.md) | High-level capability summary |
| [introduction/stakeholders.md](introduction/stakeholders.md) | Consumers and stakeholders of the service |

---

## Summary

CodeValdOrg is a **Go gRPC microservice** that manages organizational identity and access control for the CodeVald platform. It provides:

- Organization, user, role, and membership management
- Standards-based OAuth 2.0 authorization and token issuance
- Per-agency database isolation (matches CodeValdGit / CodeValdAgency)
- Admin-facing management via the CodeValdWorkOrg frontend

### Core Requirements at a Glance

| FR | Requirement |
|---|---|
| FR-001 | One Organization per Agency, stored in its own database |
| FR-002 | User lifecycle — invite, activate, suspend, remove |
| FR-003 | Role lifecycle — built-in roles + agency-defined custom roles |
| FR-004 | Membership — assign users to roles within the organization |
| FR-005 | OAuth 2.0 Authorization Code flow with PKCE (RFC 7636) |
| FR-006 | OAuth 2.0 Client Credentials flow for service-to-service auth |
| FR-007 | OAuth 2.0 token introspection (RFC 7662) and revocation (RFC 7009) |
| FR-008 | Scope-based resource permissions resolvable by the policy layer |
| FR-009 | Audit log of all identity and authorization events |
| FR-010 | Admin management surface exposed to CodeValdWorkOrg |
