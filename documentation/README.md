# CodeValdOrg — Documentation

CodeValdOrg is the **organization, identity, and access-management** microservice in the CodeVald platform.

An **Organization** is the administrative container that owns users, roles, group memberships, and OAuth 2.0 clients/tokens for a single Agency. CodeValdOrg provides OAuth 2.0 authorization flows and resource-scoped access control for the rest of the platform.

An admin interacts with CodeValdOrg via the **CodeValdWorkOrg** frontend (analogous to CodeValdWorkFrontend for CodeValdWork).

---

## Documentation Sections

| Section | Description |
|---|---|
| [1 — Software Requirements](1-SoftwareRequirements/README.md) | Requirements, introduction, problem definition, and stakeholders |
| [2 — Software Design & Architecture](2-SoftwareDesignAndArchitecture/README.md) | Architecture, data models, interfaces, OAuth flows |
| [3 — Software Development](3-SofwareDevelopment/README.md) | Research-gap analysis and (future) MVP detail docs |

---

## Quick Reference

| Item | Value |
|---|---|
| **Module** | `github.com/aosanya/CodeValdOrg` |
| **Storage** | ArangoDB — one database per agency (matches CodeValdGit / CodeValdAgency) |
| **Identity standard** | [OAuth 2.0](https://oauth.net/2/) (RFC 6749, RFC 6750, RFC 7636 PKCE) |
| **Registers with** | CodeValdCross `OrchestratorService.Register` |
| **Frontend** | [CodeValdWorkOrg](../../CodeValdWorkOrg/) — admin management UI |
