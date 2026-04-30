# Token Introspection

## Purpose

The validation path for opaque bearer tokens. Closes research-gap
**Area 3 (Token verification flow)**. Most decisions are implied by
[token-issuance.md](token-issuance.md); this doc captures the
complementary read-side semantics.

---

## Surface

Two transports, identical semantics:

| Transport | Endpoint | RFC | Used by |
|---|---|---|---|
| HTTP | `POST /{agencyId}/oauth/introspect` | RFC 7662 | External clients, browser SPAs through Cross |
| gRPC | `OrgService.Introspect` | — | In-cluster resource servers (latency-critical path; meets NFR-002 ≤ 10 ms p99) |

The HTTP and gRPC response shapes are field-for-field identical.
`scope` is a space-separated string (RFC 7662 wire format) at the HTTP
layer; the gRPC layer surfaces it as `repeated string`.

---

## Lookup algorithm

```
1. Receive token plaintext from caller.
2. Compute lookup_hash = base64url(SHA-256(plaintext)).
   (Hash covers the FULL prefixed plaintext — see token-issuance.md.)
3. Identify token kind by prefix:
      "cv_at_" → AccessToken collection
      "cv_rt_" → RefreshToken collection
      "cv_ac_" → AuthorizationCode  (introspecting auth codes is uncommon
                                     but supported per RFC 7662 §2.1)
      anything else → return {active: false}
4. Lookup entity by (agency_id, token_hash) unique index in the
   identified collection.
   Not found → return {active: false}
5. If entity.expires_at <= now → return {active: false}
6. Lookup TokenRevocation by (agency_id, token_hash):
      Found → return {active: false}
7. Return {active: true, scope, sub, client_id, exp, iat, token_type}.
```

No `crypto/subtle` constant-time compare anywhere — every comparison
is hash equality through the unique index.

---

## Response shapes

### `{active: true}` (RFC 7662 §2.2)

```json
{
  "active":     true,
  "scope":      "git:read git:write",
  "sub":        "<user-id-or-client-id-for-client-credentials-tokens>",
  "client_id":  "<oauth-client-id>",
  "exp":        1740000000,
  "iat":        1739996400,
  "token_type": "Bearer"
}
```

`sub` is the user_id when the token has an `issued_for ──▶ User`
edge; otherwise the client_id (client-credentials grant).

### `{active: false}`

```json
{ "active": false }
```

Returned for: unknown token, expired token, revoked token, malformed
token. RFC 7662 mandates returning the same minimal payload for all
of these so an attacker cannot distinguish "this token doesn't exist"
from "this token has been revoked" by response shape.

---

## Caller authentication

RFC 7662 §2.1 requires the introspection endpoint itself to be
authenticated. CodeValdOrg requires:

| Caller class | Auth method |
|---|---|
| In-cluster resource service (gRPC) | `Authorization: Bearer <client-credentials access token>` issued to that service's confidential `OAuthClient` |
| External resource server (HTTP) | Same — `Authorization: Bearer …` |
| Anonymous | Reject with `ErrInvalidClient` (`401 invalid_client`) |

Calling `Introspect` without authentication, or with a token that
doesn't carry an `org:introspect` scope, is rejected. This stops a
network adversary from probing for valid tokens.

---

## Caching contract (resource servers + Cross)

Introspection is the hot path. Naive enforcement re-introspects every
inbound request, which violates NFR-002 at scale.

The contract resource servers / Cross MAY follow:

- **Cache hit window** — cache the `{active: true}` response for
  `min(exp - now, configured_max)`. `configured_max` should be ≤ 60s.
- **Mandatory invalidation** — subscribe to
  `cross.org.{agencyID}.token.revoked`; evict the cached entry by
  `token_hash` (or by `token_id` if known) on receipt.
- **Negative caching** — `{active: false}` SHOULD NOT be cached, to
  avoid extending revocation propagation delay if a token transiently
  appears inactive (clock skew during the revoke transaction).

This contract is **opt-in for v1**. CodeValdOrg's own RPC implementation
does not cache; every call is a fresh DB lookup. See
[revocation-and-cache.md](revocation-and-cache.md) for the
full pub/sub side of the contract.

---

## Implementation notes

- `internal/server/oauthhttp.go` parses the form-encoded HTTP body
  (RFC 7662 §2.1) and dispatches to `OrgService.Introspect` for the
  shared logic.
- Introspection MUST NOT emit an `AuditEvent` on every call — the
  call rate would saturate the audit collection. Only `Introspect`
  responses with `{active: false}` due to a *revoked* token are
  audited (`event_type: token.introspect_denied`).
- The lookup is structured so a single ArangoDB round trip can fetch
  the token entity AND any matching `TokenRevocation` (via an AQL
  `LET` with two unique-index lookups). One round trip per
  introspection is the latency target.
