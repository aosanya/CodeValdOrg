# CodeValdOrg — OAuth 2.0 Protocol Flows

Sequence diagrams and invariants for every grant type. Referenced from
[architecture.md §5](architecture.md).

---

## Authorization Code + PKCE (Interactive Clients)

```
Browser                Cross (HTTP)          OrgService           Resource svc
   │                         │                   │                      │
   │── GET /authorize ───────▶                   │                      │
   │   + code_challenge      │── Authorize ──────▶                      │
   │                         │                   │                      │
   │◀── 302 redirect to login ────────────────────                      │
   │                         │                   │                      │
   │── POST /login ──────────▶                   │                      │
   │                         │── AssertUser ─────▶                      │
   │                         │                   │                      │
   │◀── 302 to redirect_uri with ?code=XYZ ───────                      │
   │                         │                   │                      │
   │── POST /token ──────────▶                   │                      │
   │   + code + verifier     │── Token ──────────▶                      │
   │                         │                   │── verify code+PKCE ──│
   │                         │                   │── consume code ──────│
   │◀── {access_token, refresh_token, exp, scope} ◀─────────────────────│
   │                         │                   │                      │
   │── GET /resource + Bearer ───────────────────────────────────────────▶
   │                         │                                          │── Introspect ──▶ OrgService
   │                         │                                          │◀── {active, scope, sub} ─│
   │◀── 200 OK ───────────────────────────────────────────────────────────│
```

Key invariants:

- `code_challenge` is persisted with the AuthorizationCode; `code_verifier` is **never** stored
- AuthorizationCode is single-use — marked consumed on first `Token` call; replay returns `invalid_grant`
- Expiry ≤ 60 seconds; enforced by a TTL index on `org_oauth_artifacts`
- Redirect URI on the `/token` call **must equal** the one supplied at `/authorize`

---

## Client Credentials (Service-to-Service)

```
Service A                              OrgService
   │── POST /token ────────────────────▶
   │   grant_type=client_credentials    │
   │   Authorization: Basic <id:secret> │
   │                                    │── verify client_secret (Argon2id)
   │                                    │── intersect requested scopes with client's allowed scopes
   │◀── {access_token, exp, scope} ─────│
```

No refresh token is issued — services re-auth when the access token expires.

---

## Refresh Token Rotation

```
Client                   OrgService
  │── POST /token ─────────▶
  │   grant_type=refresh_token,
  │   refresh_token=RT1    │── lookup RT1
  │                        │── if reused → revoke entire chain (RT1's parent, grandparent, …)
  │                        │── mint RT2 with parent=RT1; mint new access token
  │                        │── mark RT1 as consumed
  │◀── {access_token, refresh_token=RT2} ──│
```

Reuse detection: if RT1 has already been consumed when a `refresh_token` call arrives, every
RefreshToken in its rotation chain is revoked, forcing the User to re-authenticate.

---

## Introspection

Introspection is **authoritative** — resource servers never validate tokens locally in v1.

```
Resource Server                OrgService
     │── Introspect(token) ────▶
     │                           │── lookup token
     │                           │── check TTL, check revocation
     │◀── {active, scope, sub, client_id, exp, iat} ──│
```

Both HTTP (`POST /oauth/introspect`) and gRPC (`OrgService.Introspect`) surfaces exist.
In-cluster callers MUST use gRPC for the latency budget (NFR-002).

---

## Revocation

```
Client                      OrgService
  │── POST /oauth/revoke ────▶
  │   + token                  │── write TokenRevocation record
  │                            │── emit org.token.revoked event
  │◀── 200 OK ─────────────────│
```

Revocation is **synchronous** — the next `Introspect` call observes `active=false` immediately
(single-writer database per agency).
