````prompt
---
agent: agent
---

# Documentation Consistency & Organization Checker

## Purpose
Perform systematic documentation consistency checks for **CodeValdOrg**
through **one question at a time**, identifying outdated references,
consolidating related files, and ensuring the documentation structure matches
the actual implementation.

---

## Instructions for AI Assistant

Conduct a comprehensive documentation consistency analysis through **iterative
single-question exploration**. Ask ONE question at a time, wait for the
response, then decide whether to:
- **🔍 DEEPER**: Go deeper into the same topic
- **📝 NOTE**: Record an issue/gap for later action
- **➡️ NEXT**: Move to the next consistency check area
- **📊 REVIEW**: Summarise findings and determine next steps

---

## Current Technology Stack (Reference)

```yaml
Service:
  Language: Go 1.21+
  Module: github.com/aosanya/CodeValdOrg
  gRPC: google.golang.org/grpc
  Storage: ArangoDB (arangodb/go-driver)
  Crypto: golang.org/x/crypto (argon2id) + crypto/rand + crypto/sha256
  Registration: CodeValdCross OrchestratorService.Register RPC

Key interfaces:
  - OrgService: InitOrganization, GetOrganization, UpdateOrganization, DisableOrganization,
                DeleteOrganization, InviteUser, AcceptInvitation, GetUser, ListUsers,
                SuspendUser, ReactivateUser, DeleteUser, CreateRole, UpdateRole, DeleteRole,
                ListRoles, RegisterScope, DeprecateScope, ListScopes, GrantMembership,
                RevokeMembership, ListMemberships, CreateOAuthClient, RotateClientSecret,
                ListOAuthClients, DeleteOAuthClient, Authorize, Token, Introspect, Revoke,
                ListAuditEvents
  - CrossPublisher: Publish (optional — nil = events skipped in tests)

Storage collections (per agency database):
  - org_entities          — identity entities (Organization, User, PasswordCredential,
                            Role, Scope, Membership, Invitation)
  - org_oauth_clients     — OAuth client metadata (OAuthClient, ClientSecret, RedirectURI)
  - org_oauth_artifacts   — OAuth artifacts (AuthorizationCode, AccessToken, RefreshToken,
                            TokenRevocation); TTL-indexed, all immutable
  - org_audit_events      — AuditEvent; append-only, immutable
  - org_relationships     — all graph edges

Cross-service events:
  Produces: cross.org.{agencyID}.organization.created, cross.org.{agencyID}.user.invited,
            cross.org.{agencyID}.user.activated, cross.org.{agencyID}.user.suspended,
            cross.org.{agencyID}.membership.granted, cross.org.{agencyID}.token.issued,
            cross.org.{agencyID}.token.revoked
  Consumes: (none in Layer 1)

Documentation structure:
  1-SoftwareRequirements:
    requirements: documentation/1-SoftwareRequirements/requirements.md
    introduction: documentation/1-SoftwareRequirements/introduction/
  2-SoftwareDesignAndArchitecture:
    architecture: documentation/2-SoftwareDesignAndArchitecture/architecture.md
    authorization-model: documentation/2-SoftwareDesignAndArchitecture/architecture-authorization-model.md
  3-SofwareDevelopment:
    research-gaps: documentation/3-SofwareDevelopment/research-gaps.md
    mvp-details: documentation/3-SofwareDevelopment/mvp-details/
  4-QA:
    (not yet created)
```

---

## Consistency Check Areas (in priority order)

1. **Interface contract** — does `architecture.md` match actual Go interfaces in source?
2. **Data models** — do `models.go` field names match what's documented (`Key`, `Principal`, `Scope`, `AuthorizeRequest`, `Decision`)?
3. **Error types** — does `errors.go` match the error table in `architecture.md`?
4. **Registration payload** — does the `RegisterRequest` in code match the architecture doc (service name, produced topics, routes)?
5. **gRPC routes** — do the declared routes match what's actually implemented in `OrgService`?
6. **ArangoDB schema** — do the `keys` / `principals` / `scopes` collection names and indexes in code match the architecture doc?
7. **Secret-handling claims** — does the doc's promise of "plaintext returned once, never logged, never persisted" still match the code?
8. **mvp.md task status** — are completed tasks marked ✅?
9. **File size limits** — are any files over 500 lines? (run `wc -l` on suspects)

---

## Stop Conditions

- ❌ Any file in `documentation/` over 400 lines without a subfolder → **must refactor first**
- ❌ Architecture doc references interfaces that don't exist in code → **must update**
- ❌ Documentation claims about secret handling (hashing, one-time plaintext) that don't match the code → **must fix code or doc, immediately**
- ❌ `mvp.md` tasks marked 🔲 that are already implemented → **must update**
````
