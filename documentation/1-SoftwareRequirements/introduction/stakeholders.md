# Stakeholders

## Primary Consumers

| Stakeholder | Role | How They Use CodeValdOrg |
|---|---|---|
| **CodeValdCross** | Request gateway and policy enforcement point | Calls `Introspect` on every incoming request to validate bearer tokens; resolves User + scopes from the response |
| **CodeValdWorkOrg** (admin frontend) | Human admin UI | Consumes `OrgService` admin endpoints through the Cross HTTP proxy to manage Organizations, Users, Roles, Clients |
| **Every resource service** (CodeValdGit, CodeValdAgency, CodeValdWork, CodeValdAI, …) | Enforcement point | Registers its own scopes at startup; relies on the scopes in introspection responses to authorize calls |

---

## Integration Points

CodeValdOrg is called at these lifecycle events:

| Event | CodeValdOrg Call |
|---|---|
| Agency created | `OrgService.InitOrganization(agencyID, profile)` (auto, via CodeValdCross) |
| Agency deleted | `OrgService.DeleteOrganization(agencyID)` |
| Admin invites a user | `OrgService.InviteUser(agencyID, email, roles)` |
| User completes OAuth flow | `OrgService.Authorize` → `OrgService.Token` (Authorization Code + PKCE) |
| Service mints a token | `OrgService.Token` (Client Credentials) |
| Resource server validates a request | `OrgService.Introspect(token)` |
| Token revoked | `OrgService.Revoke(token)` |
| Admin lists memberships | `OrgService.ListMemberships(agencyID, filter)` |
| Audit read | `OrgService.ListAuditEvents(agencyID, filter)` |

---

## Secondary Stakeholders

| Stakeholder | Interest |
|---|---|
| **Platform operators** | Need per-agency purge and audit-log export for compliance holds and data requests |
| **Third-party integrators** | Register OAuth clients; consume the standard OAuth 2.0 surface from outside the platform |
| **AI agents (indirect)** | Acquire tokens via Client Credentials to call CodeValdGit, CodeValdWork on behalf of their service identity |
| **Human end users (indirect)** | Log in once via OAuth at the browser; don't see CodeValdOrg directly |
| **Security reviewers** | Validate OAuth 2.0 conformance, audit-log coverage, secret handling at rest |

---

## Service Maintainers

Developed and maintained alongside the rest of the CodeVald platform:

- Trunk-based development with short-lived feature branches (`feature/ORG-XXX_description`)
- Pure Go — gRPC + HTTP exposed via `cmux` on a single port (matches CodeValdGit)
- Schema-driven — `DefaultOrgSchema()` seeded through `entitygraph.SchemaManager` on startup
- Shared infrastructure via `CodeValdSharedLib` — `entitygraph`, `registrar`, `serverutil`, `arangoutil`
