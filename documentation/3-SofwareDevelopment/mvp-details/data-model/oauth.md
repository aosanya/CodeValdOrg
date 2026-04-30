# OAuth Entities

Seven entities split across two storage collections:

- `org_oauth_clients` (mutable) — `OAuthClient`, `ClientSecret`,
  `RedirectURI`
- `org_oauth_artifacts` (immutable, TTL-indexed) —
  `AuthorizationCode`, `AccessToken`, `RefreshToken`, `TokenRevocation`

---

## `OAuthClient`

Registered client — public (SPA / mobile, no secret, PKCE mandatory)
or confidential (server-side, has at least one `ClientSecret`).

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `client_id` | string | yes | unique `(agency_id, client_id)` |
| Core | `client_type` | option (`public` / `confidential`) | yes | filter index |
| Core | `name` | string | yes | — |
| Core | `allowed_grant_types` | multiselect (`authorization_code`, `client_credentials`, `refresh_token`) | yes | — |
| Long tail | `description` | string | no | — |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `updated_at` | string (RFC 3339) | no | — |
| Long tail | `disabled_at` | string (RFC 3339, nullable) | no | — |
| Long tail | `deleted_at` | string (RFC 3339, nullable) | no | — |

**Edges in:** `belongs_to_organization` (auto-inverse of
`Organization.has_oauth_client`). **Edges out:** `has_redirect_uri` (1+),
`has_client_secret` (1+ — confidential clients only),
`allows_scope` (1+ → `Scope`).

`allowed_grant_types` is the only place a `MultiSelect` is acceptable
because the value set is fixed. Redirect URIs and allowed scopes are
modelled as edges, not as list properties.

---

## `ClientSecret`

Mirrors the `PasswordCredential` split — confidential clients can have
multiple `ClientSecret`s to support rotation grace. Created by
`CreateOAuthClient` and `RotateClientSecret`; old secrets retain a
`grace_expires_at` window during which they are still honoured.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `secret_hash` | string (Argon2id PHC) | yes | — |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `last_used_at` | string (RFC 3339) | no | — |
| Long tail | `revoked_at` | string (RFC 3339, nullable) | no | — |
| Long tail | `grace_expires_at` | string (RFC 3339, nullable) | no | — |

**Edges in:** `belongs_to_oauth_client` (auto-inverse of
`OAuthClient.has_client_secret`).

The plaintext secret is returned exactly once at the moment of
creation or rotation (NFR-004) — never re-derivable.

---

## `RedirectURI`

Each pre-registered redirect URI is its own entity, chosen over a
JSON-serialised array in Q13 of the Q&A. Independent metadata
per URI (`last_used_at`, `revoked_at`) and per-URI revocation become
trivial.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `uri` | string | yes | — (uniqueness within OAuthClient enforced service-side) |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `last_used_at` | string (RFC 3339) | no | — |
| Long tail | `revoked_at` | string (RFC 3339, nullable) | no | — |

**Edges in:** `belongs_to_oauth_client` (auto-inverse of
`OAuthClient.has_redirect_uri`).

Exact-string match at `/authorize` and `/token` (architecture §1 — no
wildcards). Service-layer enforces uniqueness of `uri` within a parent
OAuthClient.

---

## `AuthorizationCode`

`Immutable: true`. Single-use code, ≤ 60 s TTL, bound to `client_id`,
`redirect_uri`, and PKCE challenge. Lives in `org_oauth_artifacts`.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `code_hash` | string (SHA-256, base64url) | yes | unique `(agency_id, code_hash)` |
| Core | `code_challenge` | string (PKCE S256 challenge) | yes | — |
| Core | `redirect_uri` | string (denormalised — must match `/token` request) | yes | — |
| Core | `expires_at` | string (RFC 3339, ≤ 60 s in the future) | yes | TTL index |
| Long tail | `state` | string | no | — |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `consumed_at` | string (RFC 3339, nullable — non-null on first `/token` call) | no | — |

**Edges out:** `issued_to ──▶ OAuthClient`, `issued_for ──▶ User`,
`has_requested_scope ──▶ Scope` (one per requested scope).

`code_verifier` is **never** persisted; only the challenge is stored.
Replay returns `invalid_grant`.

---

## `AccessToken`

`Immutable: true`. Opaque server-generated bearer token. v1
introspection is the only validity check; v2 may layer signed JWTs.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `token_hash` | string (SHA-256, base64url) | yes | unique `(agency_id, token_hash)` |
| Core | `expires_at` | string (RFC 3339) | yes | TTL index |
| Long tail | `created_at` | string (RFC 3339) | no | — |

**Edges out:** `issued_to ──▶ OAuthClient`, `issued_for ──▶ User`
(absent for the `client_credentials` grant), `has_scope ──▶ Scope`
(effective scopes — the intersection computed at issuance).

Token plaintext is high-entropy random bytes (256-bit) returned to
the client once; only the hash is stored. Lookup is hash equality —
no `crypto/subtle` compare needed.

---

## `RefreshToken`

`Immutable: true`. Rotating, single-use. The `parent` self-edge tracks
the rotation chain so reuse detection can revoke every ancestor.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `token_hash` | string (SHA-256, base64url) | yes | unique `(agency_id, token_hash)` |
| Core | `expires_at` | string (RFC 3339) | yes | TTL index |
| Long tail | `created_at` | string (RFC 3339) | no | — |
| Long tail | `consumed_at` | string (RFC 3339, nullable; non-null = rotated) | no | — |

**Edges out:** `issued_to ──▶ OAuthClient`, `issued_for ──▶ User`,
`has_scope ──▶ Scope`, `parent ──▶ RefreshToken` (rotation chain).

Reuse detection: if a `/token` call presents an already-consumed
`RefreshToken`, traverse `parent` ancestors and revoke every member of
the chain (`refresh_reuse` reason).

---

## `TokenRevocation`

`Immutable: true`. The kill-record for a still-unexpired access or
refresh token. Introspection does a parallel hash-lookup against this
collection; the record self-purges via TTL once the underlying token
would have naturally expired.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `token_hash` | string (the hash of the revoked token) | yes | unique `(agency_id, token_hash)` |
| Core | `revoked_at` | string (RFC 3339) | yes | — |
| Core | `expires_at` | string (RFC 3339, matches the underlying token's natural expiry) | yes | TTL index |
| Long tail | `reason` | option (`user_logout` / `admin_revoke` / `refresh_reuse` / `disable_org` / `suspend_user`) | no | — |

**No edges.** The lookup is hash equality on `token_hash`; no
traversal to the revoked token entity is needed. Cheaper at
introspection time and matches the architecture §4.5 principle that
revocation is a separate status lookup, not a mutation on the token
artefact itself.
