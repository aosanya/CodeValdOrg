# CodeValdOrg — Authorization Model

## Decision

**v1 uses a distributed Policy Decision Point (PDP).** CodeValdOrg
owns identity and token issuance only; every resource service is its
own PDP, comparing the requested operation against the `scope[]` claim
returned by `Introspect`. There is **no** `OrgService.Authorize` RPC.

Decided 2026-04-27 during the research-gap Q&A. Marked "for now" — see
the **Re-visit triggers** at the bottom.

## Status

ACCEPTED for v1. Closes research-gap **Area 5 (Authorization decision
flow)**.

---

## What this means concretely

```
Browser ──Bearer abc123─────▶ Resource service (e.g. CodeValdGit)
                                      │
                                      │── Introspect(abc123) ──▶ CodeValdOrg
                                      │◀── {active: true, scope: ["git:read","git:write"], sub: "user-42"} ──│
                                      │
                                      │── compares requested op ("PushRef") against scope[] LOCALLY
                                      │
                              ◀── 200 / 403 ─
```

- The decision logic — "does `git:write` cover `PushRef`?" — runs **inside CodeValdGit**, not CodeValdOrg.
- CodeValdOrg's surface for authorization is `Introspect`. Nothing more.
- The resource service is responsible for keeping its scope-to-action mapping current as it adds new operations.

## What CodeValdOrg owns

| Concern | Where it lives |
|---|---|
| Identity (who is calling) | CodeValdOrg — `User`, `OAuthClient` |
| Authentication (proof of identity) | CodeValdOrg — Authorization Code + PKCE, Client Credentials, refresh-token rotation |
| Issuance (minting the bearer token) | CodeValdOrg — `Token` endpoint computes effective scope set as `min(requested, user grant via roles, client allowed)` |
| Validity (is the token still good?) | CodeValdOrg — `Introspect` checks expiry + revocation |
| Decision (is *this* call allowed?) | **Resource service** (distributed PDP) |

## What CodeValdOrg explicitly does **not** own (in v1)

- No `Authorize(principal, scope, resource)` RPC
- No fine-grained per-resource ABAC ("can user X edit *this specific* commit?") — out of scope
- No central scope-to-action mapping table — each resource service owns its own
- No central deny-list enforcement at request time — the only "deny" CodeValdOrg can express is `Introspect` returning `active: false`

---

## Trade-offs accepted

| Pro | Con |
|---|---|
| Single network hop on the hot path (`Introspect` only) — meets NFR-002 p99 ≤ 10 ms with room to spare | Scope-to-action logic duplicates across every resource service |
| Resource teams own the policy nearest the resource — no cross-team coupling on scope semantics | Easy to drift: two services interpret `git:write` differently |
| CodeValdOrg stays small and protocol-pure — it's an OAuth 2.0 AS, not a policy engine | No central audit of "who was allowed to do what" — only "who held what scope" |
| `Introspect` response is cacheable in-cluster (Cross can cache for the token's remaining TTL or until revocation event) | Revocation propagation becomes load-bearing — see research-gap Area 6 |

The trade-off is acceptable in v1 because (a) the platform is small
enough that scope-to-action mappings can be reviewed across services
in PR review, and (b) we have not yet found a use-case that requires
fine-grained ABAC. The decision should be re-visited if either
condition flips.

---

## Re-visit triggers

The "for now" stays in force until at least one of these is true:

1. **A resource service needs a per-instance decision** ("can user X
   edit commit `abc123`?") that requires CodeValdOrg state which the
   resource server doesn't (and shouldn't) cache. Adding fine-grained
   ABAC to resource services would force them to read CodeValdOrg
   data — at that point centralising the decision is the smaller
   change.
2. **Scope-string drift causes a security incident.** Two services
   interpreting the same scope differently is the classic distributed-PDP
   failure mode. One incident is the trigger for centralisation.
3. **A regulatory requirement** demands a single audit trail of every
   authorisation decision (not just every token issuance). Distributed
   PDPs make this expensive to assemble after the fact.

When triggered, evolve to the **hybrid model** that was option (c) in
Q17: keep `Introspect` for the common scope-string case, add
`OrgService.Authorize(token, action, resource_id)` for fine-grained
checks. This is additive — no breaking changes to existing resource
servers.

---

## Implications for the rest of the architecture

- **`OrgService` interface** (architecture.md §6) stays as-is — no
  `Authorize` method.
- **Scope grammar** (research-gap Area 4) is the contract between
  CodeValdOrg's issuance and the resource servers' enforcement.
  Whatever grammar we choose has to be unambiguous enough that
  independent services interpret it the same way.
- **Introspection caching** (research-gap Area 6) becomes the
  performance pressure point — distributed PDPs amplify the call rate
  on `Introspect`.
- **Audit log** (FR-009) records issuance events but not enforcement
  events. If a resource service denies a request, it is its own
  responsibility to log that locally — CodeValdOrg never sees it. This
  is a known coverage gap and is the price of the distributed model.
