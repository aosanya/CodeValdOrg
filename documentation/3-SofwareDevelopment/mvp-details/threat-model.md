# Threat Model

## Purpose

STRIDE-shaped threat analysis for CodeValdOrg, broken down by
attacker class. Closes research-gap **Area 13 (Threat model)** and
supersedes the high-level mitigation table in
[architecture.md §9](../../2-SoftwareDesignAndArchitecture/architecture.md).

---

## Attacker classes

| # | Class | Capabilities | Out of scope |
|---|---|---|---|
| A1 | Network adversary on the public internet | Can probe public OAuth endpoints; can intercept TLS-terminated requests in transit (defended by TLS at ingress) | Cannot read service memory; cannot read DB |
| A2 | Compromised end-user device | Can replay session cookies; can read storage of an SPA / mobile client; can hijack the browser tab | Cannot impersonate other users; cannot read service memory |
| A3 | Malicious OAuth client developer | Has registered a public OAuthClient; controls a redirect URL; can craft any client-side request | Cannot read other clients' secrets; cannot bypass scope intersection |
| A4 | Compromised resource server (insider service) | Holds a confidential OAuthClient and a valid `org:introspect` scope; can issue many `Introspect` calls; can cache responses | Cannot mint tokens; cannot write to other agencies' DBs |
| A5 | Operator with read-only DB access | Can query ArangoDB collections directly | Cannot write to the DB; cannot read service memory |
| A6 | Operator with read/write DB access | Can read & modify ArangoDB collections (i.e. has dropped to direct AQL) | — (this is a privileged role; treated as trusted) |

A6 is **trusted by design**. The threat model is concerned with
A1–A5; A6 is the operator with admin DB credentials and is governed
by deployment access control, not by CodeValdOrg's runtime.

---

## STRIDE per class

Each cell either names the existing mitigation (with a section pointer)
or marks the threat as **OPEN** (intentional or not yet covered).

### A1 — Network adversary

| STRIDE | Threat | Mitigation |
|---|---|---|
| **S**poof | Forge `Authorization: Bearer …` | Tokens are 256-bit random; SHA-256 lookup; unguessable. |
| **T**amper | Replay an intercepted authorization code | Single-use + ≤ 60 s TTL + bound to PKCE verifier ([token-issuance.md](token-issuance.md)) |
| **R**epudiate | Deny having issued a `/token` call | `AuditEvent` logs `actor_id`, `subject_id`, `source_ip`, `outcome` (FR-009) |
| **I**nfo disclosure | Probe `/oauth/introspect` for valid tokens | Endpoint requires authentication ([introspection.md](introspection.md)); anonymous calls → 401 `invalid_client` |
| **D**enial of service | Flood `/token` with invalid client_id | In-process per-`client_id` token-bucket limiter; per-IP fallback for `/authorize` where no client is presented yet. See [Rate-limiting policy](#rate-limiting-policy) below. |
| **E**levation | Inject a chosen `redirect_uri` (open-redirect) | Exact-string match against pre-registered `RedirectURI` (architecture §5.1) |

### A2 — Compromised end-user device

| STRIDE | Threat | Mitigation |
|---|---|---|
| **S**poof | Steal session, impersonate user from a different device | Refresh tokens are rotated single-use; reuse from a different session triggers chain revocation ([token-issuance.md](token-issuance.md)) |
| **T**amper | Modify `code_verifier` mid-flow | Cryptographically bound to the stored `code_challenge`; mismatch → `invalid_grant` |
| **R**epudiate | "Someone else used my account" | `AuditEvent` ties events to `user_id` + `source_ip` + `source_user_agent` |
| **I**nfo disclosure | Read `localStorage` to steal access token | **Accepted residual for v1** — public-client tokens live in the browser; v1 defence is short access-token TTL (1 h default) + refresh-rotation reuse detection. v2 will add DPoP token binding (RFC 9449); see [v2 backlog](#mitigations-not-yet-covered-v2-backlog). |
| **D**oS | Browser tab spam `/authorize` | Same as A1 — in-process per-IP limiter on `/authorize`. See [Rate-limiting policy](#rate-limiting-policy). |
| **E**levation | Try to upgrade scope via refresh | Rejected — refresh-token effective scope is bounded by the original (`token-issuance.md` step 3) |

### A3 — Malicious OAuth client developer

| STRIDE | Threat | Mitigation |
|---|---|---|
| **S**poof | Pretend to be a different OAuthClient | `client_id` + `client_secret` (Argon2id) verification; public clients have no secret but are bound by the registered redirect-URI list |
| **T**amper | Register a `redirect_uri` that proxies the auth code to the attacker | Limited blast radius — the attacker can only steal codes destined for *their own* client_id. Cannot affect other clients. |
| **R**epudiate | "I never registered that redirect URI" | `AuditEvent` for every `oauth_client.created` / `oauth_client.rotated` |
| **I**nfo disclosure | Request scopes a client isn't entitled to | Effective scope intersection rejects: `requested ∩ client.allowedScopes`. Excess scopes silently dropped. Empty result → `invalid_scope`. |
| **D**oS | Spam `/token` with malformed bodies | Metric-counter only ([error-catalog.md](error-catalog.md) Q22) — does not pollute the audit log. Per-`client_id` rate limit applies once the malformed-body burst is large; see [Rate-limiting policy](#rate-limiting-policy). |
| **E**levation | Try to use Authorization Code with no PKCE | PKCE mandatory for public clients ([scope-model.md](scope-model.md), architecture §5.1); reject |

### A4 — Compromised resource server

This is the most concerning class because the resource server holds a
legitimate token-issuing client.

| STRIDE | Threat | Mitigation |
|---|---|---|
| **S**poof | Use its `Introspect` capability to mint tokens for itself | `Introspect` reads only — does not mint. Resource server has no `Token`-mint authority. |
| **T**amper | Modify cached `{active: true}` responses to extend a revoked token's life | **PARTIALLY MITIGATED** — pub/sub revocation invalidates caches ([revocation-and-cache.md](revocation-and-cache.md)); resource server cooperation is required. A truly malicious cache could ignore revocation events. Defence is the strict cache TTL ceiling (60 s) — the worst-case revocation propagation delay. |
| **R**epudiate | Deny a denial decision | The resource server's own audit; not CodeValdOrg's responsibility (distributed-PDP trade-off — see [architecture-authorization-model.md](../../2-SoftwareDesignAndArchitecture/architecture-authorization-model.md)) |
| **I**nfo disclosure | Mass-`Introspect` to probe for valid tokens | Token entropy is 256-bit; cannot be guessed. Volume can be detected by metric counter on `Introspect` rate. |
| **D**oS | Flood `Introspect` and exhaust the per-agency DB | Caching contract MUST be honoured ([introspection.md](introspection.md)); abusive callers detectable via `metric-count`. In-process per-`client_id` limiter on `/oauth/introspect` (1000 req/s default). |
| **E**levation | Cross-agency probe (call `Introspect` for a different `agency_id`) | Server cross-checks `agency_id` against baked-in `AGENCY_ID` ([configuration.md](configuration.md)); mismatch → 403 |

### A5 — Operator with read-only DB access

| STRIDE | Threat | Mitigation |
|---|---|---|
| **S**poof | Use a read AccessToken hash to mint a fake token | Cannot — needs the plaintext, which is never stored |
| **T**amper | n/a (read-only) | — |
| **R**epudiate | n/a (no actions) | — |
| **I**nfo disclosure | Read `password_hash`, `secret_hash` cold | Argon2id PHC; computationally infeasible to invert at sensible passwords. Recommend operator-level access control as defence in depth. |
| **D**oS | n/a | — |
| **E**levation | Cannot escalate without write access | — |

---

## Rate-limiting policy

Resolved 2026-04-27 in Q31. CodeValdOrg runs an **in-process
token-bucket limiter** keyed by `client_id` (or source IP for
`/authorize` where no client is presented yet). Single-agency-per-process
([configuration.md](configuration.md) Q24) means per-process state is
the right granularity; no Redis needed.

| Endpoint | Key | Default sustained | Default burst | Env var to override |
|---|---|---|---|---|
| `/oauth/token` | `client_id` | 50 req/s | 100 | `ORG_RATELIMIT_TOKEN_PER_CLIENT` |
| `/oauth/authorize` | source IP | 20 req/s | 40 | `ORG_RATELIMIT_AUTHORIZE_PER_IP` |
| `/oauth/introspect` | `client_id` | 1000 req/s | 2000 | `ORG_RATELIMIT_INTROSPECT_PER_CLIENT` |
| Admin RPCs (Cross-proxied) | `client_id` | 100 req/s | 200 | `ORG_RATELIMIT_ADMIN_PER_CLIENT` |

Implementation: `golang.org/x/time/rate.Limiter` per key in an LRU
cache (10 000-entry default; oldest entry evicted on cache pressure).
Memory footprint is bounded.

When the limit is exceeded:

| Surface | Response |
|---|---|
| HTTP / OAuth | `429 Too Many Requests` + body `{"error": "temporarily_unavailable", "error_description": "rate limit exceeded"}` (RFC 6749 §5.2 — `temporarily_unavailable` is the closest fit; some implementations use a non-standard `rate_limited`, but RFC-defined is safer) |
| gRPC | `codes.ResourceExhausted` |
| Logs | `info` level, includes `client_id` or source IP, no stack |

Rate-limit hits emit no `AuditEvent` — they are pure operational
signals; the `metric-count` handling label applies (see
[error-catalog.md](error-catalog.md) — `ErrRateLimitExceeded`).

The limiter is the FIRST gate in the request pipeline (before
authentication for `/authorize`; right after `client_id` extraction
for everything else). This keeps abusive traffic from reaching the
DB.

## Mitigations not yet covered (v2 backlog)

### DPoP token binding — committed for v2

[RFC 9449 — Demonstrating Proof of Possession](https://datatracker.ietf.org/doc/html/rfc9449)
binds an access token to a client-held private key. Every request to a
resource server MUST present a `DPoP` header — a JWS signed with that
private key, covering the HTTP method + URL + a nonce. A stolen access
token alone becomes useless without the key.

Why it's the right mitigation for the A2 info-disclosure residual:

- **Browser SPAs hold the key in a non-extractable `CryptoKey`** via
  the WebCrypto API (`crypto.subtle.generateKey({extractable: false})`).
  The key cannot be exfiltrated by JS, even with full XSS. Only the
  *use* of the key — signing one DPoP proof at a time — is exposed.
- **Mobile and CLI clients hold the key in OS keychain / Secure
  Enclave / TPM.** Same property: the key never leaves the device.
- **Defeats verbatim token theft.** An attacker who exfiltrates the
  access token from `localStorage` cannot use it because they cannot
  produce a matching DPoP proof.

Scope of the v2 work:

| Surface | Change |
|---|---|
| `/token` | Accept `DPoP` header on the token request; bind the public-key thumbprint (`jkt`) into the issued AccessToken via a new `cnf.jkt` field on the entity. Reject requests where the same `jkt` reuses a `jti` (DPoP-proof replay). |
| `/oauth/introspect` | Return `cnf: {jkt: "..."}` in the introspection response when the token is DPoP-bound. |
| Resource servers | MUST verify the inbound `DPoP` header against the introspection response's `cnf.jkt` before authorising the call. |
| `OAuthClient` | New `dpop_required` flag (default `false` for backward compat; SPAs and mobile clients flip to `true`). |
| `AccessToken` schema | Add `cnf_jkt` long-tail property (nullable; non-null = DPoP-bound). |

Compatibility note: DPoP is **additive**. Bearer-only flows continue
to work for clients (e.g. server-to-server confidential clients) that
don't need the binding. v2 introduces DPoP without breaking v1
clients.

Tracking: this entry is the v2 commitment for A2 info disclosure;
when the work begins it gets its own `architecture-dpop.md` design
document.

### Other v2 backlog
- **Multi-factor authentication** — v1 is password-only; MFA is layered on top in v2.
- **Webhook signing** for `cross.org.{agencyID}.token.revoked` events — currently relies on the in-cluster trust boundary; v2 may sign payloads so external SIEM can verify.
- **Anomaly detection** on the `metric-count` series (e.g. spike in `invalid_grant_total` from one `client_id` triggers admin alert) — operational, not in the service.

---

## Accepted residual risks

These are **explicitly accepted** for v1; they are not gaps to close
later, they are deliberate trade-offs.

| Risk | Reason |
|---|---|
| Browser-resident access tokens (A2 disclosure) — **v1 only** | OAuth 2.0 + PKCE is the industry standard for v1. DPoP is a committed v2 deliverable (see [v2 backlog](#dpop-token-binding--committed-for-v2)); BFF is an alternative deployment pattern documented elsewhere |
| Distributed-PDP scope drift across resource services | One incident is the trigger to centralise (see [architecture-authorization-model.md](../../2-SoftwareDesignAndArchitecture/architecture-authorization-model.md) re-visit triggers) |
| Audit log records issuance, not enforcement decisions | Distributed-PDP trade-off — same authorization-model doc |
| A6 (operator with DB write) is fully trusted | Standard operational boundary; defended by deployment-time access control, not by service runtime |

---

## How this document evolves

- Each new attack scenario (CVE, internal red-team finding, customer
  report) gets a row added to the appropriate STRIDE table — never
  silently fixed without a record.
- Mitigations marked **OPEN** are tracked here, not in a separate TODO
  file. Closing one means filling the cell in with a section pointer.
- v2 features (DPoP, MFA, rate limiting) get their own
  `architecture-{topic}.md` document AND a row update here.
