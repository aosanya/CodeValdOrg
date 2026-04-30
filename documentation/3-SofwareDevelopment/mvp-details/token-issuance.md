# Token Issuance

## Purpose

The end-to-end recipe for minting OAuth artifacts (`AuthorizationCode`,
`AccessToken`, `RefreshToken`). Closes research-gap **Area 2 (Token
issuance flow)**. Decided 2026-04-27 in Q26–Q27 of the research Q&A.

Builds on:

- [data-model/oauth.md](data-model/oauth.md) — entity shapes and edges
- [scope-model.md](scope-model.md) — effective-scope intersection
- [error-catalog.md](error-catalog.md) — failure-mode mapping

---

## Token plaintext format

Every token CodeValdOrg issues to a caller is a **prefixed** base64url
string:

| Token kind | Prefix | Plaintext shape | Length |
|---|---|---|---|
| `AuthorizationCode` | `cv_ac_` | `cv_ac_<base64url(32 random bytes)>` | 49 chars |
| `AccessToken` | `cv_at_` | `cv_at_<base64url(32 random bytes)>` | 49 chars |
| `RefreshToken` | `cv_rt_` | `cv_rt_<base64url(32 random bytes)>` | 49 chars |
| `Invitation` token | `cv_iv_` | `cv_iv_<base64url(32 random bytes)>` | 49 chars |

The 32 random bytes come from `crypto/rand.Read` — 256 bits of
entropy, more than enough that brute-force is irrelevant. The prefix
is **scanned-by-leak-detectors** convention (Stripe / GitHub style):
automated tooling can grep CI logs, repos, and pastebins for `cv_at_`
and trigger revocation.

---

## Token hash at rest

Stored in the entity's `token_hash` (or `code_hash` / `token_hash` for
invitations) property:

```
token_hash = base64url( SHA-256( full_prefixed_plaintext ) )
```

Critical: the hash covers the **full prefixed string**, not just the
32-byte random tail. Two reasons:

1. **Defence in depth** against a bug that queries the wrong
   collection — `SHA-256("cv_at_xyz")` ≠ `SHA-256("cv_rt_xyz")` even
   for the same random tail, so a misdirected lookup returns nothing
   instead of a wrong entity.
2. **Future-proofing** — if we ever change the prefix scheme (e.g.
   add an environment marker like `cv_test_at_…`), old hashes
   automatically don't collide with new ones.

Lookup at validation time is by **hash equality** — no
`crypto/subtle` constant-time compare needed because we never compare
two plaintexts; we compute the hash and look up by indexed equality.

Hash algorithm is **SHA-256, not Argon2id**. Tokens are high-entropy
random bytes (256-bit) — there is nothing to brute-force, so a fast
hash is correct. Argon2id is reserved for user-chosen secrets
(passwords, client secrets) where computational cost is the defence.

---

## Authorization Code + PKCE flow

Triggered by `Token` RPC with `grant_type = authorization_code`.

```
1.  Validate request:
      - client_id known and not deleted/disabled         → ErrInvalidClient
      - code is a valid `cv_ac_…` plaintext              → ErrInvalidGrant
      - redirect_uri matches AuthorizationCode.redirect_uri exactly  → ErrRedirectURIMismatch
      - SHA-256(code_verifier) == AuthorizationCode.code_challenge → ErrPKCEMismatch
      - AuthorizationCode.consumed_at IS NULL            → ErrInvalidGrant (replay)
      - AuthorizationCode.expires_at > now               → ErrInvalidGrant (expired)

2.  Compute effective scope (see scope-model.md):
      effective = AuthorizationCode.requested_scopes
                ∩ scopesGrantedToUser
                ∩ client.allowedScopes
                ∩ {s : s.deprecated_at == null}
      effective empty → ErrInvalidScope

3.  Generate AccessToken plaintext:
      access_plain = "cv_at_" + base64url(crypto/rand 32 bytes)

4.  Generate RefreshToken plaintext (only if client allows refresh_token):
      refresh_plain = "cv_rt_" + base64url(crypto/rand 32 bytes)

5.  Persist:
      - CreateEntity AccessToken {token_hash: SHA256(access_plain), expires_at: now + ORG_ACCESS_TOKEN_TTL, ...}
      - Edges: issued_to → OAuthClient, issued_for → User, has_scope → Scope (×N for effective)
      - CreateEntity RefreshToken {token_hash: SHA256(refresh_plain), expires_at: now + ORG_REFRESH_TOKEN_TTL, ...}  (if applicable)
      - Edges: issued_to → OAuthClient, issued_for → User, has_scope → Scope (×N for effective)
      - UpdateEntity AuthorizationCode {consumed_at: now}
      - All four operations in a single ArangoDB transaction.

6.  Publish:
      - cross.org.{agencyID}.token.issued event with token IDs (NOT plaintext)
      - Publish failure → see "Failure-mode contract" below.

7.  Return to caller:
      {
        "access_token":  access_plain,
        "refresh_token": refresh_plain,    // omitted if not minted
        "token_type":    "Bearer",
        "expires_in":    seconds-until-AccessToken.expires_at,
        "scope":         space-separated effective scope names
      }

8.  Audit: AuditEvent {event_type: "token.issued", outcome: "success",
                       actor_id: client_id, subject_id: user_id, payload: {scopes: [...]}}.
```

---

## Client Credentials flow

Triggered by `Token` RPC with `grant_type = client_credentials`.

```
1.  Validate request:
      - Authorization: Basic <client_id>:<client_secret> OR
        Authorization: Basic / form params client_secret_post
      - Look up OAuthClient by client_id                         → ErrInvalidClient
      - client.client_type == "confidential"                     → ErrInvalidClient
      - Argon2id verify client_secret against active ClientSecret OR
        any ClientSecret with grace_expires_at > now             → ErrInvalidClient
      - client.allowed_grant_types contains "client_credentials" → ErrUnauthorizedClient

2.  Compute effective scope (no user; grants come from the client):
      effective = requested ∩ client.allowedScopes ∩ {not deprecated}
      effective empty → ErrInvalidScope

3.  Generate AccessToken plaintext (same as step 3 above).

4.  Persist:
      - CreateEntity AccessToken — issued_for edge is ABSENT (no user).
      - Edges: issued_to → OAuthClient, has_scope → Scope (×N).
      - NO RefreshToken — client-credentials never issues one (FR-006).

5.  Publish: cross.org.{agencyID}.token.issued (same as step 6 above).

6.  Return: same shape as Authorization Code, minus refresh_token.

7.  Audit: AuditEvent {event_type: "token.issued", outcome: "success",
                       actor_id: client_id, subject_id: client_id, ...}.
```

---

## Refresh Token rotation flow

Triggered by `Token` RPC with `grant_type = refresh_token`.

```
1.  Validate request:
      - Look up RefreshToken by SHA-256 lookup           → ErrInvalidGrant
      - RefreshToken.expires_at > now                    → ErrInvalidGrant
      - issued_to edge → OAuthClient matches request's client_id → ErrInvalidGrant

2.  Reuse detection:
      IF RefreshToken.consumed_at IS NOT NULL:
        # The presented token has already been rotated; the chain has been compromised.
        - Walk parent ancestors via parent edge.
        - For every RefreshToken in the chain: write TokenRevocation with reason = "refresh_reuse".
        - For the AccessToken issued alongside the original mint and any descendants: same.
        - Publish cross.org.{agencyID}.token.revoked for each.
        - Return ErrInvalidGrant.

3.  Compute effective scope:
      Default = the original RefreshToken.has_scope set (no expansion possible).
      If `scope` parameter present, effective = requested ∩ original ∩ {not deprecated}; cannot be a superset.
      Empty → ErrInvalidScope.

4.  Mint new pair:
      - access_plain = "cv_at_" + base64url(rand 32)
      - refresh_plain = "cv_rt_" + base64url(rand 32)

5.  Persist (single transaction):
      - CreateEntity AccessToken (new)
      - CreateEntity RefreshToken (new) with parent → presented RefreshToken
      - Copy issued_to, issued_for, has_scope edges from original onto both new entities
      - UpdateEntity presented RefreshToken {consumed_at: now}

6.  Publish: cross.org.{agencyID}.token.issued.

7.  Return: same shape as Authorization Code.

8.  Audit: AuditEvent {event_type: "token.refreshed", outcome: "success", ...}.
```

---

## Failure-mode contract — persistence vs publish

Decided in Q26. Token mint is a strong-CP operation: both the DB write
and the Cross publish must succeed, otherwise the request fails.

```
                  ┌─ DB write OK ──┬─ Publish OK ──→ Return token, audit success
                  │                │
Mint request ────┤                │
                  │                └─ Publish FAILS → Delete just-written entities
                  │                                  (compensating delete; best-effort)
                  │                                  Return 503 ErrTemporarilyUnavailable
                  │
                  └─ DB write FAILS ──→ Return 503 ErrTemporarilyUnavailable
                                        (publish never attempted)
```

**Compensating-delete failure is harmless.** If the post-publish-fail
delete itself fails, an orphan token row exists in the DB. The
plaintext was never returned to the caller, so its hash is
unguessable; the token is unreachable. The TTL index purges the
orphan when its `expires_at` passes. Operations may also choose to
run a periodic sweep for tokens with no surviving `issued_for` /
`issued_to` edges, but it's not required for correctness.

This is the strong-consistency choice (Q26 option b). Alternatives
explicitly rejected:

- **Best-effort publish** — risks Cross-cache silent drift.
- **Outbox pattern** — defers the publish via a worker; available as a
  v2 evolution if the strong-CP coupling becomes operationally painful.

---

## Cross events — payload shape

All token-issuance publishes go to topic
`cross.org.{agencyID}.token.issued`:

```json
{
  "event_id":     "<uuid>",
  "event_at":     "2026-04-27T12:34:56Z",
  "agency_id":    "<agency-id>",
  "client_id":    "<oauth-client-id>",
  "user_id":      "<user-id-or-null-for-client-credentials>",
  "token_id":     "<entity-_key-of-the-AccessToken>",
  "token_kind":   "access",
  "scopes":       ["git:read", "git:write"],
  "expires_at":   "2026-04-27T13:34:56Z"
}
```

Plaintext is **never** in the event payload. Subscribers (Cross cache,
audit aggregator) deal in IDs and metadata only.

---

## Resolved implementation items

- **`crypto/rand` failure handling.** Extremely rare on Linux but
  possible on exhausted entropy in test environments or hardware-RNG
  failure. The mint function returns `ErrTemporarilyUnavailable`
  (which maps to `503 temporarily_unavailable` per
  [error-catalog.md](error-catalog.md)). The underlying error from
  `crypto/rand.Read` is logged at `error` level with the request
  correlation ID; never surfaced in the response body. Callers (OAuth
  clients) receive standard 503 retry semantics. No fallback PRNG
  is used — if the OS entropy source is broken, refusing to mint is
  correct.
- **Clock skew on `expires_at`.** Assumed bounded by NTP. No
  per-request skew handling; resource servers compare against their
  own clock at introspection time. The `Clock` interface
  ([testing-strategy.md](testing-strategy.md)) is the only
  abstraction over `time.Now()` and is for test determinism, not
  skew compensation.
- **`cv_` prefix collision.** Reserved-by-convention. The
  `cv_<2-letter-kind>_` namespace is owned by CodeValdOrg; future Org
  or sibling-service token kinds must pick a non-clashing 2-letter
  kind (currently used: `ac` / `at` / `rt` / `iv`).
